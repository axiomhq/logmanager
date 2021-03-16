package logmanager

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	axMaxBatchSize  = 1000
	axFlushDuration = time.Second
)

func init() {

}

type Event struct {
	Time     string `json:"_time,omitempty"`
	Level    string `json:"level,omitempty"`
	Module   string `json:"module,omitempty"`
	Filename string `json:"filename,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message,omitempty"`
	Process  string `json:"process,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// AxiomDBWriter will write out to a console
type AxiomDBWriter struct {
	sync.Mutex
	events      []*Event
	client      *http.Client
	baseURL     string
	authToken   string
	dataset     string
	debug       bool
	processName string
	hostname    string
}

// NewAxiomDBWriter ...
func NewAxiomDBWriter() *AxiomDBWriter {
	w := &AxiomDBWriter{
		dataset:     "axiom-logs",
		debug:       false,
		processName: path.Base(os.Args[0]),
	}
	w.hostname, _ = os.Hostname()

	dbURL := os.Getenv("AXIOM_DEBUG_AXIOMDB_URL")
	if dbURL == "" {
		return w
	}
	if _, err := url.Parse(dbURL); err != nil {
		fmt.Println("Unable to parse database URL:", err)
		return w
	}

	w.baseURL = dbURL
	w.authToken = os.Getenv("AXIOM_DEBUG_AXIOMDB_TOKEN")

	w.client = &http.Client{
		Timeout: time.Second * 30,
	}

	dataset := os.Getenv("AXIOM_DEBUG_DATASET")
	if dataset != "" {
		w.dataset = dataset
	}

	if os.Getenv("AXIOM_DEBUG_AXIOMDB_DEBUG") != "" {
		w.debug = true
	}

	go w.postBatch()
	return w
}

// BuildTheme ...
func (w *AxiomDBWriter) BuildTheme(module string) ColorTheme {
	return ColorTheme{}
}

// Log ...
func (w *AxiomDBWriter) Log(level Level, theme ColorTheme, module, filename string, line int, timestamp time.Time, message string) {
	if w.client == nil {
		return
	}

	event := &Event{}
	event.Time = timestamp.In(time.UTC).Format(time.RFC3339Nano)
	event.Filename = filepath.Base(filename)
	event.Level = level.String()
	event.Module = module
	event.Line = line
	event.Message = message
	event.Process = w.processName
	event.Hostname = w.hostname

	w.Lock()
	w.events = append(w.events, event)
	w.Unlock()
}

func (w *AxiomDBWriter) postBatch() {
	if w.client == nil {
		return
	}

	dataURL := w.baseURL
	if strings.HasSuffix(dataURL, "/") {
		dataURL = dataURL[:len(dataURL)-1]
	}
	dataURL += fmt.Sprintf("/datasets/%s/ingest", w.dataset)

	for {
		time.Sleep(axFlushDuration)

		w.Lock()
		batch := w.events
		w.events = []*Event{}
		w.Unlock()

		if len(batch) == 0 {
			continue
		}

		data, err := json.Marshal(batch)
		if err != nil {
			if w.debug {
				fmt.Println("Unable to create event:", err)
			}
			continue
		}

		req, err := http.NewRequest(http.MethodPost, dataURL, gzipStream(bytes.NewReader(data)))
		if err != nil {
			if w.debug {
				fmt.Println("Unable to create HTTP request:", err)
			}
			// no point in trying again, will most likely fail again
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		if w.authToken != "" {
			req.Header.Set("Authorization", "Bearer "+w.authToken)
		}

		resp, err := w.client.Do(req)
		if err != nil {
			if w.debug {
				fmt.Println("Unable to send to axiomdb:", err)
			}

			w.prependEvents(batch)
			continue
		}

		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			// yey
		case http.StatusServiceUnavailable:
			if w.debug {
				fmt.Println("Error from axiom:", resp.Status)
			}
			w.prependEvents(batch)
		default:
			if w.debug {
				fmt.Println("Error from axiom:", resp.Status)
			}
		}
	}
}

func (w *AxiomDBWriter) prependEvents(batch []*Event) {
	w.Lock()
	defer w.Unlock()

	w.events = append(batch, w.events...)
	if len(w.events) > axMaxBatchSize {
		w.events = w.events[len(w.events)-axMaxBatchSize:]
	}
}

func gzipStream(r io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func(r io.Reader) {
		defer pw.Close()

		// Does not fail when using a predefined compression level.
		gzw, _ := gzip.NewWriterLevel(pw, gzip.BestSpeed)
		defer gzw.Close()

		if _, err := io.Copy(gzw, r); err != nil {
			fmt.Fprintf(os.Stderr, "error compressing data to ingest: %s\n", err)
		}
	}(r)

	return pr
}
