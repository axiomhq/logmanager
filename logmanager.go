package logmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Envvars
var (
	loggerSpec = os.Getenv("AXIOM_DEBUG")
)

// Level defines the log level, from Trace to Critical
type Level uint64

// Set ...
func (l *Level) Set(newLevel Level) { atomic.StoreUint64((*uint64)(l), (uint64)(newLevel)) }

// Get ...
func (l *Level) Get() Level { return (Level)(atomic.LoadUint64((*uint64)(l))) }

// String ...
func (l *Level) String() string {
	switch l.Get() {
	case Trace:
		return "trace"
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warning:
		return "warn"
	case Error:
		return "error"
	case Critical:
		return "crit"
	default:
		return "unspecified"
	}
}

// Levels
const (
	Trace Level = iota
	Debug
	Info
	Warning
	Error
	Critical
)

var (
	globalWriters     = []Writer{NewConsoleWriter()}
	customWritersSet  uint64
	globalWritersLock sync.RWMutex
)

// SetCustomWriters ...
func SetCustomWriters(writers ...Writer) {
	if atomic.LoadUint64(&customWritersSet) > 0 {
		panic(errors.New("custom writers has already been set"))
	}

	globalWritersLock.Lock()
	defer globalWritersLock.Unlock()
	// copy, so we can "atomically" replace globalWriters
	newWriters := append([]Writer{}, writers...)
	globalWriters = newWriters
	atomic.StoreUint64(&customWritersSet, 1)
}

// getWriters will return the current global writers if the given src is nil
func getWriters() []Writer {
	globalWritersLock.RLock()
	defer globalWritersLock.RUnlock()
	return globalWriters
}

// ColorTheme ...
type ColorTheme struct {
	Module string
	Levels []string
}

// Writer defines something that accepts log input, it is expected to send it somewhere
type Writer interface {
	Log(level Level, theme ColorTheme, module string, filename string, line int, timestamp time.Time, message string)
	BuildTheme(module string) ColorTheme
}

// Logger is a logmanager base logger
type Logger struct {
	name  string
	level Level

	themeGenerated   uint32
	writeDescriptors []writeDescriptor
	writeDescLock    *sync.RWMutex
}

type writeDescriptor struct {
	writer Writer
	theme  ColorTheme
}

// GetLogger will get a logger for the specified name
func GetLogger(name string) Logger {
	// go through the loggerSpec and look for "Foo=Trace" or whatever
	// it will always prefer longer module names that match but also match smaller ones
	// for example, for module name foo.bar.baz, foo.bar=Trace will match but foo.bar.baz=Trace will be prefered
	// it's not smart enough to figure out that foo.bar.ba should be ignored in preference of foo.bar
	// but that is a compromise that i'm willing to make for simplicity and flexability
	level := Info
	levelHint := ""
	for _, moduleInfo := range strings.Split(loggerSpec, ":") {
		splitted := strings.SplitN(moduleInfo, "=", 2)
		if len(splitted) != 2 {
			continue
		}

		moduleName := splitted[0]
		moduleLevel := splitted[1]

		if len(moduleName) <= len(levelHint) {
			continue
		}

		if strings.HasPrefix(moduleName, name) || moduleName == "<root>" {
			switch {
			case strings.EqualFold(moduleLevel, "trace"):
				level = Trace
			case strings.EqualFold(moduleLevel, "info"):
				level = Info
			case strings.EqualFold(moduleLevel, "debug"):
				level = Debug
			case strings.EqualFold(moduleLevel, "warning"):
				level = Warning
			case strings.EqualFold(moduleLevel, "error"):
				level = Error
			case strings.EqualFold(moduleLevel, "critical"):
				level = Critical
			default:
				println("Warning: couldn't understand moduleName/loggerLevel", moduleName, moduleLevel)
				continue
			}
			levelHint = moduleName
		}
	}

	return Logger{
		name:          name,
		level:         level,
		writeDescLock: &sync.RWMutex{},
	}
}

// IsDebugEnabled will return if debug is enabled, for this specific logger.
// note this is a bad mechanism to detect a general debug build state. for that you should use build flags
func (l *Logger) IsDebugEnabled() bool {
	return l.level.Get() <= Debug
}

// LogLevel will return the current log level
func (l *Logger) LogLevel() Level {
	return l.level.Get()
}

// SetLogLevel will set the given log level
func (l *Logger) SetLogLevel(level Level) {
	l.level.Set(level)
}

// Logger logging methods

// Trace ...
func (l *Logger) Trace(message string, args ...interface{}) { l.Log(Trace, message, args...) }

// Debug ...
func (l *Logger) Debug(message string, args ...interface{}) { l.Log(Debug, message, args...) }

// Info ...
func (l *Logger) Info(message string, args ...interface{}) { l.Log(Info, message, args...) }

// Warn ...
func (l *Logger) Warn(message string, args ...interface{}) { l.Log(Warning, message, args...) }

// Critical ...
func (l *Logger) Critical(message string, args ...interface{}) { l.Log(Critical, message, args...) }

// Error ...
func (l *Logger) Error(message string, args ...interface{}) error {
	l.Log(Error, message, args...)
	if len(args) > 0 {
		if err, ok := args[0].(error); ok {
			return err
		} else if err, ok := args[len(args)-1].(error); ok {
			return err
		}
	}

	return fmt.Errorf(message, args...)
}

func (l *Logger) buildDescriptors() {
	writers := getWriters()

	// should only be true on the first .Log
	// init the theme for each writer for this package
	l.writeDescLock.Lock()
	for _, writer := range writers {
		desc := writeDescriptor{writer: writer, theme: writer.BuildTheme(l.name)}
		l.writeDescriptors = append(l.writeDescriptors, desc)
	}
	l.writeDescLock.Unlock()
}

// Log ...
func (l *Logger) Log(level Level, message string, args ...interface{}) {
	// safe if not called before writers are added
	if atomic.CompareAndSwapUint32(&l.themeGenerated, 0, 1) == true {
		l.buildDescriptors()
	}

	if level < l.level {
		return
	}

	ts := time.Now().UTC()
	_, filepath, line, ok := runtime.Caller(2)
	if ok == true {
		filepath = path.Base(filepath)
	} else {
		filepath = "__unknown__"
		line = -1
	}

	msg := fmt.Sprintf(message, args...)
	l.writeDescLock.RLock()
	for _, desc := range l.writeDescriptors {
		desc.writer.Log(level, desc.theme, l.name, filepath, line, ts, msg)
	}
	l.writeDescLock.RUnlock()
}

// JSONify will attempt to jsonify the given structure
// useful for debugging
// shorthand for encoding/json marshalling but handling errors and conversion to string
func (l *Logger) JSONify(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("ERRMARSHALLING=%s", err)
	}
	return string(b)
}

// JSONifyIndent will attempt to jsonify the given structure - with opinionated indenting
// useful for debugging
// shorthand for encoding/json marshalling but handling errors and conversion to string
func (l *Logger) JSONifyIndent(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("ERRMARSHALLING=%s", err)
	}
	return string(b)

}

// IsError is a good replacement for if err != nil {
// you can use if log.IsError(err) { and avoid having to write
// the same error logging code again and again.
// also much more useful error logging
func (l *Logger) IsError(err error) bool {
	if err == nil {
		return false
	} else {
		msg := SPrintStack(3, 8)
		msg += fmt.Sprintf("Detected error: %v\n", err)
		l.Log(Error, msg)

		return true
	}
}

type stringer interface {
	String() string
}

// Recover is intended to be used when recovering from panics, it will function similiarly to IsError
// in that it will output a stacktrace, but it has additional handling to strip out the panic stack
// err is interface{} for convenience as that is what you will receieve from recover()
// example:
//
//      func mycode() (err error) {
//          defer func() {
//				if r := logger.Recover(recover()); r != nil{
//					err = r
//				}
//			}()
//          panic(errors.New("oh no"))
//          return nil
//      }
//
// in addition Recover will covert any string(er) type to an error if an error is not found
func (l *Logger) Recover(v interface{}) error {
	if v == nil {
		return nil
	}

	var err error
	switch verr := v.(type) {
	case error:
		err = verr
	case string:
		err = errors.New(verr)
	case stringer:
		err = errors.New(verr.String())
	}

	if err == nil {
		err = fmt.Errorf("Unknown panic error: %v", v)
	}

	msg := SPrintStack(5, maxCallers)
	msg += fmt.Sprintf("Detected panic: %v\n", err)
	l.Log(Error, msg)

	return err
}

// CheckErr will close closer when called, and print an error
// if one is returned. This function is intended to be used in
// defer scenarios, where the error would otherwise be lost
func (l *Logger) CheckErr(f func() error) {
	l.IsError(f())
}

// PrintStackTrace will print the current stack out to the info logger channel
func (l *Logger) PrintStackTrace() {
	l.Log(Info, SPrintStack(3, maxCallers))
}

// PrintCaller prints caller of this function
func (l *Logger) PrintCaller(skip int) {
	l.Log(Info, SPrintCaller(skip+2))
}

// columnedLines takes care of formatting columned output
// that is, if you want to have three columns of values all lined up, this will line them up
type columnedLines struct {
	columnValues [][]string
	columnSizes  []int
}

func (c *columnedLines) Add(columns ...string) {
	c.columnValues = append(c.columnValues, columns)

	for i, v := range columns {
		if len(c.columnSizes) <= i {
			c.columnSizes = append(c.columnSizes, len(v))
			continue
		}

		if vsize := len(v); c.columnSizes[i] < vsize {
			c.columnSizes[i] = vsize
		}
	}
}

func (c *columnedLines) ColumnString(columns []string) (line string) {
	for i, column := range columns {
		size := c.columnSizes[i]
		line += column
		if padding := size - len(column); padding > 0 {
			line += strings.Repeat(" ", padding)
		}
	}
	return
}

func (c *columnedLines) String() (lines string) {
	for _, columns := range c.columnValues {
		lines += c.ColumnString(columns) + "\n"
	}
	return
}

func (c *columnedLines) Reverse() (lines string) {
	for i := len(c.columnValues) - 1; i >= 0; i-- {
		columns := c.columnValues[i]
		lines += c.ColumnString(columns) + "\n"
	}
	return
}

const maxCallers = 50
const cutDir = "watchly/src"

// SPrintCaller returns a string with information about caller
func SPrintCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "Cannot determine caller"
	}
	msg := "Caller:\n"

	if srcDir := strings.Index(file, cutDir); srcDir != -1 {
		file = "..." + file[srcDir+len(cutDir):]
	}

	fn := runtime.FuncForPC(pc)
	fnName := "unknown"
	if fn != nil {
		fnName = fn.Name()
	}

	msg += fmt.Sprintf("  %s:L%d  @ %s", file, line, fnName)
	return msg
}

// SPrintStack will print the current stack to a returned string
func SPrintStack(skip, max int) string {
	if max > maxCallers {
		max = maxCallers
	}

	var msg string
	msg = ""
	msg += fmt.Sprintf("Trace (most recent call first, max %d stack):\n", max)

	// we allocate a lot of callers because we don't know how many there are, so we just get a lot
	callers := make([]uintptr, maxCallers)
	totalCallers := runtime.Callers(skip, callers[:])
	callers = callers[:totalCallers]

	columns := columnedLines{}
	frames := runtime.CallersFrames(callers)
	foundFrames := 0
	more := true
	for more == true && foundFrames <= max {
		var frame runtime.Frame
		frame, more = frames.Next()

		// skip some packages because they are uninteresting
		if strings.HasPrefix(frame.Function, "runtime") ||
			strings.HasPrefix(frame.Function, "testing") ||
			strings.HasPrefix(frame.Function, "reflect") ||
			strings.Contains(frame.Function, "github.com/stretchr/testify") {
			continue
		}

		file := frame.File
		if srcDir := strings.Index(file, cutDir); srcDir != -1 {
			file = "..." + file[srcDir+len(cutDir):]
		}

		columns.Add(fmt.Sprintf("  %s:%d", file, frame.Line), fmt.Sprintf(" @ %s", frame.Function))
		foundFrames++
	}
	return msg + columns.String()
}
