package logmanager

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

// DiskWriterConfig ...
type DiskWriterConfig struct {
	RotateDuration  time.Duration // rotates after time.Duration since log file creation date
	MaximumLogFiles int
}

// DiskWriter ...
type DiskWriter struct {
	DiskWriterConfig
	logpath string

	logbuf chan string
}

// rotateLogs will rotate the current logs and return the next rotation time
// nolint
func (w DiskWriter) rotateLogs() (time.Time, error) {
	if _, err := os.Stat(w.logpath); err != nil {
		// no log to rotate yet
		os.MkdirAll(path.Dir(w.logpath), 0777)
		if f, err := os.Create(w.logpath); err != nil {
			return time.Time{}, err
		} else {
			f.Close()
			return time.Now().Add(w.RotateDuration), nil
		}
	}

	logfiles := []string{path.Base(w.logpath)}
	logDir := path.Dir(w.logpath)
	// find all the log files that currently exist
	for i := 1; ; i++ {
		logfilename := fmt.Sprintf("%s.%d", path.Base(w.logpath), i)
		logpath := path.Join(logDir, logfilename)
		_, err := os.Stat(logpath)
		if err != nil {
			break
		}

		logfiles = append(logfiles, logfilename)
	}

	// reverse traverse the list so we can rename foo.log.3 before foo.log.2
	for i := len(logfiles) - 1; i >= 0; i-- {
		oldName := logfiles[i]
		newName := fmt.Sprintf("%s.%d", path.Base(w.logpath), i+1)

		os.Rename(path.Join(logDir, oldName), path.Join(logDir, newName))

		if i >= w.MaximumLogFiles-1 {
			os.Remove(path.Join(logDir, newName))
		}
	}

	os.MkdirAll(path.Dir(w.logpath), 0777)
	if f, err := os.Create(w.logpath); err != nil {
		return time.Time{}, err
	} else {
		f.Close()
		return time.Now().Add(w.RotateDuration), nil
	}
}

func writeAll(writer io.Writer, buf []byte) error {
	for len(buf) > 0 {
		n, err := writer.Write(buf)
		buf = buf[n:]
		if err != nil {
			return err
		}
	}

	return nil
}

// NewDiskWriter ...
func NewDiskWriter(logpath string, config DiskWriterConfig) *DiskWriter {
	w := &DiskWriter{config, logpath, make(chan string, 10000)}
	go func() {
		var err error
		var file *os.File
		rotateTime := time.Time{}

		for logLine := range w.logbuf {
			// rotate the logs if it has been longer than w.RotateDuration since last rotation
			if time.Now().After(rotateTime) {
				rotateTime, err = w.rotateLogs()
				if err != nil {
					println("Warning, could not create logfile:", err.Error())
					continue
				}
				if file != nil {
					file.Close()
					file = nil
				}
			}

			if file == nil {
				file, err = os.OpenFile(w.logpath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
				if err != nil {
					println("Warning, could not open logfile for appending:", err.Error())
					continue
				}
			}

			// write into the log file
			err = writeAll(file, []byte(logLine))
			if err != nil {
				println("Warning, Error writing logfile:", err.Error())
			}
		}

		if file != nil {
			file.Close()
		}
	}()

	return w
}

// Close will end the writer
func (w *DiskWriter) Close() {
	close(w.logbuf)
}

// BuildTheme ...
func (w *DiskWriter) BuildTheme(string) ColorTheme {
	return ColorTheme{}
}

// Log ...
func (w *DiskWriter) Log(level Level, _ ColorTheme, module, filename string, line int, timestamp time.Time, message string) {
	if level <= Debug {
		return
	}

	ts := timestamp.In(time.UTC).Format("15:04:05")
	filename = filepath.Base(filename)
	select {
	case w.logbuf <- fmt.Sprintf("%s %s %s %s:%d %s\n", ts, level.String(), module, filename, line, message):
	default:
		println("WARNING: could not log to logfile, buffer full")
	}
}
