package logmanager

import (
	"fmt"
	"hash/crc32"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

func init() {
	// the library detects whether the terminal supports color, override with this
	switch os.Getenv("AXIOM_COLORED_OUTPUT") {
	case "1":
		color.NoColor = false
	case "0":
		color.NoColor = true
	}

	pName := path.Base(os.Args[0])
	if strings.HasPrefix(os.Args[0], path.Join(os.TempDir(), "go-build")) && !strings.HasSuffix(os.Args[0], ".test") {
		// check to see if the executing program is /tmp/go-build<numbers>/whatever
		// which generally means this is a go run type thing
		// in this case the pName we can get is just a long annoying hash
		// so change it to a shorthash
		if len(pName) > 8 {
			pName = pName[:8]
		}
	}
	processName = getColor(pName)(pName)
}

// colors
var (
	processName  string
	moduleColors = []func(string, ...interface{}) string{
		color.New(color.FgHiGreen, color.Faint).SprintfFunc(),
		color.New(color.FgHiGreen).SprintfFunc(),
		color.New(color.FgGreen).SprintfFunc(),
		color.New(color.FgYellow, color.Faint).SprintfFunc(),
		color.New(color.FgHiYellow).SprintfFunc(),
		color.New(color.FgYellow).SprintfFunc(),
		color.New(color.FgHiBlue, color.Faint).SprintfFunc(),
		color.New(color.FgHiBlue).SprintfFunc(),
		color.New(color.FgBlue).SprintfFunc(),
		color.New(color.FgHiMagenta, color.Faint).SprintfFunc(),
		color.New(color.FgHiMagenta).SprintfFunc(),
		color.New(color.FgMagenta).SprintfFunc(),
		color.New(color.FgHiCyan, color.Faint).SprintfFunc(),
		color.New(color.FgHiCyan).SprintfFunc(),
		color.New(color.FgCyan).SprintfFunc(),
		color.New(color.FgWhite, color.Faint).SprintfFunc(),
	}
)

func getColor(str string) func(string, ...interface{}) string {
	hash := crc32.ChecksumIEEE([]byte(str))
	return moduleColors[int(hash)%len(moduleColors)]
}

// ConsoleWriter will write out to a console
type ConsoleWriter struct {
}

// NewConsoleWriter ...
func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{}
}

// BuildTheme ...
func (w *ConsoleWriter) BuildTheme(module string) ColorTheme {
	moduleColor := getColor(module)
	return ColorTheme{
		Module: moduleColor(module),
		Levels: []string{
			color.New(color.FgWhite).SprintFunc()("trace"),
			color.New(color.FgGreen).SprintFunc()("debug"),
			color.New(color.FgBlue).SprintFunc()("info "),
			color.New(color.FgYellow).SprintFunc()("warn "),
			color.New(color.FgRed).SprintFunc()("error"),
			color.New(color.BgRed).SprintFunc()("critc"),
		},
	}
}

// Log ...
func (w *ConsoleWriter) Log(level Level, theme ColorTheme, module, filename string, line int, timestamp time.Time, message string) {
	ts := timestamp.In(time.UTC).Format("15:04:05.00")
	filename = filepath.Base(filename)

	fmt.Printf("[%s] %s %s@%s %s:%d %s\n", ts, theme.Levels[level], processName, theme.Module, filename, line, message)
}
