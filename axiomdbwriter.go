package logmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	axiomdb "axicode.axiom.co/watchmakers/axiomdb/client"
)

var (
	axClient        *axiomdb.Client
	axDataset       = "axiom-logs"
	axDebug         = false
	axMaxBatchSize  = 1000
	axFlushDuration = time.Second
)

func init() {
	url := os.Getenv("AXIOM_DEBUG_AXIOMDB_URL")
	if url == "" {
		return
	}

	var err error
	axClient, err = axiomdb.NewClient(url)
	if err != nil {
		fmt.Println("Unable to connect to axiomdb:", err)
		return
	}

	dataset := os.Getenv("AXIOM_DEBUG_DATASET")
	if dataset != "" {
		axDataset = dataset
	}

	if os.Getenv("AXIOM_DEBUG_AXIOMDB_DATASET") != "" {
		axDebug = true
	}
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
	events []*Event
}

// NewAxiomDBWriter ...
func NewAxiomDBWriter() *AxiomDBWriter {
	w := &AxiomDBWriter{}
	go w.postBatch()
	return w
}

// BuildTheme ...
func (w *AxiomDBWriter) BuildTheme(module string) ColorTheme {
	return ColorTheme{}
}

// Log ...
func (w *AxiomDBWriter) Log(level Level, theme ColorTheme, module, filename string, line int, timestamp time.Time, message string) {
	if axClient == nil {
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
			if axDebug {
				fmt.Println("Unable to create event:", err)
			}
			continue
		}

		_, err = axClient.Datasets.Ingest(context.Background(), axDataset, bytes.NewReader(data), axiomdb.JSON, axiomdb.Identity, axiomdb.IngestOptions{})
		if err != nil {
			if axDebug {
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
