package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	validator "github.com/SpecterOps/chow/pkg/validator"
)

func main() {
	if len(os.Args) < 2 {
		slog.Error("no files provided")
		os.Exit(1)
	}

	fileName := os.Args[1]

	reader, err := os.Open(fileName)
	if err != nil {
		slog.Error("failed to open file", "fileName", fileName, "error", err)
		os.Exit(1)
	}

	jsonSchema, err := validator.LoadIngestSchema()
	if err != nil {
		slog.Error("failed to load ingest schema", "error", err)
		os.Exit(1)
	}

	reader.Seek(0, io.SeekStart)

	start := time.Now()

	decoder := json.NewDecoder(reader)
	v := validator.NewValidator(decoder, jsonSchema)

	report, err := v.Validate()
	if err != nil {
		slog.Error("failed to validate", "err", err)
	}

	fmt.Printf("%+v\n", report)

	slog.Info("Completed", "time", time.Since(start))
}
