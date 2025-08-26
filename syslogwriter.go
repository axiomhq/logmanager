package logmanager

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const rfc5424 = "2006-01-02T15:04:05.000Z07:00"

var utf8bom = []byte{0xef, 0xbb, 0xbf}

// Most of this code imitates the golang syslog implementation
// but as a lot of that is hard-coded to use specific (poor) formatting
// we re-implement

// Taken from the go stdlib implementation of syslog
// modified to not export values
const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	_logEmerg int = iota
	_logAlert
	logCrit
	logErr
	logWarning
	_logNotice
	logInfo
	logDebug
)

// a sub-interface of net.Conn, makes mocking simpler in tests
type connwriter interface {
	Close() error
	LocalAddr() net.Addr
	Write(b []byte) (n int, err error)
}

// unixSyslog opens a connection to the syslog daemon running on the
// local machine using a Unix domain socket.
func unixSyslog() (conn connwriter, err error) {
	logTypes := []string{"unixgram", "unix"}
	logPaths := []string{"/dev/log", "/var/run/syslog", "/var/run/log"}
	for _, network := range logTypes {
		for _, path := range logPaths {
			conn, err := net.Dial(network, path)
			if err != nil {
				continue
			}
			return conn, nil
		}
	}
	return nil, errors.New("unix syslog delivery error")
}

// SyslogWriter ...
type SyslogWriter struct {
	m sync.Mutex

	isClosed  uint32
	conn      connwriter
	localConn bool
	hostname  string
	network   string
	raddr     string

	bufferedMessages chan string
	minLevel         Level
}

// NewSyslogWriter returns a writer that will send log messages to a syslog server
// configured with the provied network, raddr strings.
// Passing in "" as the network will default to using default unix sockets
func NewSyslogWriter(network, raddr string) (*SyslogWriter, error) {
	w := SyslogWriter{
		network: network,
		raddr:   raddr,

		bufferedMessages: make(chan string, 1000),
		minLevel:         Warning,
	}

	if err := w.connect(); err != nil {
		return nil, err
	}

	go w.sendloop()
	return &w, nil
}

func (w *SyslogWriter) connect() (err error) {
	w.m.Lock()
	defer w.m.Unlock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}

	if w.network == "" {
		if w.hostname == "" {
			w.hostname = "localhost"
		}
		w.localConn = true
	}

	var conn connwriter
	if w.network == "" {
		conn, err = unixSyslog()
	} else {
		conn, err = net.Dial(w.network, w.raddr)
	}

	if err == nil {
		w.conn = conn
		if w.hostname == "" {
			w.hostname = conn.LocalAddr().String()
		}
	}
	return
}

func (w *SyslogWriter) sendloop() {
	// intended to be a long running goroutine
	// uses a slice buffer and sync.Cond to syncronise instead of a channel
	// because a channel has a limited capacity which may cause log messages
	// to block at some point (if network goes down, there is likely to be a spike
	// in log messages that can not be sent)

	var backoffCounter int32

	for {
		ok := w.sendBufferedMessages()
		if ok {
			// network connection problem, backoff for a while to stop any hammering
			if backoffCounter < 7 {
				backoffCounter++
			}

			<-time.After((time.Millisecond * 50) * time.Duration(rand.Int31n(backoffCounter))) //nolint:gosec // This is fine here.
		}

		for len(w.bufferedMessages) < 1 {
			if atomic.LoadUint32(&w.isClosed) == 1 {
				// connection closed, we need to exit
				return
			}
		}
	}
}

func (w *SyslogWriter) sendBufferedMessages() (ok bool) {
	ok = true

	for message := range w.bufferedMessages {
		err := w.sendMessage(message)
		if err != nil {
			ok = false
			break
		}
	}
	if ok {
		defer w.connect() //nolint:errcheck // If it fails, it fails...
	}
	return
}

func (w *SyslogWriter) sendMessage(message string) error {
	_, err := w.conn.Write([]byte(message))
	return err
}

// BuildTheme ...
func (w *SyslogWriter) BuildTheme(_ /*module*/ string) ColorTheme { return ColorTheme{} }

// Log ...
func (w *SyslogWriter) Log(level Level, _ ColorTheme, module, filename string, line int, timestamp time.Time, message string) {
	if atomic.LoadUint32(&w.isClosed) == 1 {
		return
	}

	var priority int

	switch {
	case level < w.minLevel:
		return
	case level == Debug:
		priority = logDebug
	case level == Info:
		priority = logInfo
	case level == Warning:
		priority = logWarning
	case level == Error:
		priority = logErr
	case level == Critical:
		priority = logCrit
	}

	hostname := "-"
	if w.localConn {
		hostname = w.hostname
	}

	header := fmt.Sprintf("<%d>1 %s %s %s %d - -", priority, timestamp.Format(rfc5424), hostname, module, os.Getpid())
	msg := fmt.Sprintf("%s %s%s:%d %s\n", header, utf8bom, filename, line, message)

	select {
	case w.bufferedMessages <- msg:
	default:
		println("Syslog-logger Warning: too many messages buffered, syslog losing messages")
	}
}
