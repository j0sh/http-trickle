package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"trickle"
)

// Listens to new streams from MediaMTX and publishes
// to trickle HTTP server under the same name
// then concurrently subscribes and writes to an outfile

// TODO correctly handle shutdown - shut down publisher and ffmpeg

var (
	baseURL    *string
	streamName *string
)

type SegmentPoster struct {
	tricklePublisher *trickle.TricklePublisher
}

func (sp *SegmentPoster) NewSegment(reader io.Reader) {
	go func() {
		// NB: This blocks! Very bad!
		sp.tricklePublisher.Write(reader)
	}()
}

func segmentPoster(streamName string) *SegmentPoster {
	c, err := trickle.NewTricklePublisher(*baseURL + "/" + streamName)
	if err != nil {
		panic(err)
	}
	return &SegmentPoster{
		tricklePublisher: c,
	}
}

func runPublish(streamName string) {
	slog.Info("Starting stream", "stream", streamName)

	r, w, err := os.Pipe()
	if err != nil {
		slog.Error("Error acquriing r/w pipes", "stream", streamName, "err", err)
		os.Exit(1)
	}
	defer w.Close()

	go func() {
		sp := segmentPoster(streamName)
		defer sp.tricklePublisher.Close()
		(&trickle.MediaSegmenter{
			ExtraFiles: []*os.File{r},
		}).RunSegmentation(fmt.Sprintf("pipe:%d", r.Fd()), sp.NewSegment)
		slog.Info("Completing publish", "stream", streamName)
	}()

	if _, err := io.Copy(w, os.Stdin); err != io.EOF {
		slog.Error("Error copying", "stream", streamName, "err", err)
	}
}

func main() {

	// Check some command-line arguments
	baseURL = flag.String("url", "http://localhost:2939", "Base URL for the stream")
	streamName = flag.String("stream", "", "Output stream name (required)")
	flag.Parse()
	if *streamName == "" {
		log.Fatalf("Error: Output stream name is required. Use -stream flag.")
	}
	runPublish(*streamName)
}
