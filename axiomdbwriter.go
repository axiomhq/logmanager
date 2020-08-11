package logmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	axiomdb "axicode.axiom.co/watchmakers/axiomdb/client"
)

var (
	axClient  *axiomdb.Client
	axDataset = "axiom-logs"
	axDebug   = false
)

func init() {
	url := os.Getenv("AXIOMDB_URL")
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

	if os.Getenv("AXIOM_DEBUG_AXIOMDB") != "" {
		axDebug = true
	}
}

// AxiomDBWriter will write out to a console
type AxiomDBWriter struct {
}

// NewAxiomDBWriter ...
func NewAxiomDBWriter() *AxiomDBWriter {
	return &AxiomDBWriter{}
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

	event := map[string]interface{}{}
	event["_time"] = timestamp.In(time.UTC).Format(time.RFC3339Nano)
	event["filename"] = filepath.Base(filename)
	event["level"] = level.String()
	event["module"] = module
	event["line"] = line
	event["message"] = message

	data, err := json.Marshal([]map[string]interface{}{event})
	if err != nil {
		if axDebug {
			fmt.Println("Unable to create event:", err)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// FIXME: AxiomDB client should do some internal buffering of events before sending
	_, err = axClient.Datasets.Ingest(ctx, axDataset, bytes.NewReader(data), axiomdb.JSON, axiomdb.Identity, axiomdb.IngestOptions{})
	if err != nil {
		if axDebug {
			fmt.Println("Unable to send to axiomdb:", err)
		}
		return
	}
}
