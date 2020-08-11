package logmanager

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	axiomdb "axicode.axiom.co/watchmakers/axiomdb/client"
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
}

// AxiomDBWriter will write out to a console
type AxiomDBWriter struct {
	sync.Mutex
	events  []*Event
	client  *axiomdb.Client
	dataset string
	debug   bool
}

// NewAxiomDBWriter ...
func NewAxiomDBWriter() *AxiomDBWriter {
	w := &AxiomDBWriter{
		dataset: "axiom-logs",
		debug:   false,
	}

	url := os.Getenv("AXIOM_DEBUG_AXIOMDB_URL")
	if url == "" {
		return w
	}

	var err error
	w.client, err = axiomdb.NewClient(url)
	if err != nil {
		fmt.Println("Unable to connect to axiomdb:", err)
		return w
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

	w.Lock()
	w.events = append(w.events, event)
	w.Unlock()
}

func (w *AxiomDBWriter) postBatch() {
	if w.client == nil {
		return
	}

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

		_, err = w.client.Datasets.Ingest(context.Background(), w.dataset, gzipStream(bytes.NewReader(data)), axiomdb.JSON, axiomdb.GZIP, axiomdb.IngestOptions{})
		if err != nil {
			if w.debug {
				fmt.Println("Unable to send to axiomdb:", err)
			}

			w.Lock()
			w.events = append(batch, w.events...)
			if len(w.events) > axMaxBatchSize {
				w.events = w.events[len(w.events)-axMaxBatchSize:]
			}
			w.Unlock()
			continue
		}
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
