package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"trickle"
)

// TrickleSubscriber example - write segments to file

func main() {

	// Check some command-line arguments
	baseURL := flag.String("url", "http://localhost:2939", "Base URL for the stream")
	streamName := flag.String("stream", "", "Stream name (required)")
	flag.Parse()
	if *streamName == "" {
		slog.Error("Error: stream name is required. Use -stream flag to specify the stream name.")
		return
	}

	client := trickle.NewTrickleSubscriber(*baseURL + "/" + *streamName)

	for {
		// Read and process the first segment
		resp, err := client.Read()
		if err != nil {
			if errors.Is(err, trickle.EOS) {
				slog.Info("End of stream signal")
				return
			}
			slog.Error("Error reading subscription", "err", err)
			return
		}
		idx := trickle.GetSeq(resp)
		fname := fmt.Sprintf("out-read/%d.ts", idx)
		file, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		n, err := io.Copy(file, resp.Body)
		if err != nil {
			slog.Error("Failed to record segment", "seq", idx, "err", err)
			return
		}
		resp.Body.Close()
		file.Close()
		slog.Info("--- End of Segment ", "seq", idx, "bytes", trickle.HumanBytes(n))
	}
	slog.Info("Completing", "stream", *streamName)
}
