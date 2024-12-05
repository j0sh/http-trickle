package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"trickle"
)

// publishes a stream taken from stdin

var (
	baseURL    *string
	streamName *string
)

type SegmentPoster struct {
	tricklePublisher *trickle.TricklePublisher
	ffmpegWriter     *os.File
}

func (sp *SegmentPoster) NewSegment(reader io.Reader) {
	go func() {
		// NB: This blocks! Very bad!
		if err := sp.tricklePublisher.Write(reader); err != nil {
			slog.Error("Error writing trickle", "err", err)
			if err == trickle.StreamNotFoundErr {
				slog.Error("Trickle stream not found")
				sp.ffmpegWriter.Close()
			}
			if err != nil {
				io.Copy(io.Discard, reader)
			}
		}
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

	// Listen for the interrupt signal to initiate a graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	wg, doneCh := WaitGroupWithDone()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sp := segmentPoster(streamName)
		sp.ffmpegWriter = w
		defer sp.tricklePublisher.Close()
		(&trickle.MediaSegmenter{
			ExtraFiles: []*os.File{r},
		}).RunSegmentation(fmt.Sprintf("pipe:%d", r.Fd()), sp.NewSegment)
		slog.Info("Completing publish", "stream", streamName)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(w, os.Stdin); !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
			slog.Error("Error copying", "stream", streamName, "err", err)
		}
		w.Close() // Close to stop the segmenter
	}()

	// Wait for either a signal or the completion of io.Copy
	select {
	case <-sigs:
		slog.Info("Received interrupt, shutting down...")
		w.Close() // Close the write end of the pipe to stop the io.Copy
		wg.Wait()
	case <-doneCh:
	}
	slog.Info("Stopped stream", "stream", streamName)
}

// WaitGroupWithDone returns a `*sync.WaitGroup` and a `doneChannel`
// that will be closed once all goroutines have completed.
func WaitGroupWithDone() (*sync.WaitGroup, <-chan struct{}) {
	wg := &sync.WaitGroup{}
	done := make(chan struct{})

	// This goroutine waits for all tasks to finish and then closes the channel
	go func() {
		wg.Wait()   // Block until WaitGroup counter is zero
		close(done) // Close the channel to signal completion
	}()

	return wg, done
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
