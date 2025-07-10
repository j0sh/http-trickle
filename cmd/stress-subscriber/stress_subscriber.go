package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	"trickle" // adjust import path as needed
)

func main() {
	// ---- CLI flags ---------------------------------------------------------
	baseURL := flag.String("url", "http://localhost:2939", "Base URL of the stream server")
	stream := flag.String("stream", "", "Stream base name (required)")
	count := flag.Int("count", 1, "Number of concurrent streams to subscribe to")
	flag.Parse()

	if *stream == "" {
		slog.Error("Error: stream name is required. Use -stream flag to specify the stream name.")
		os.Exit(1)
	}
	if *count < 1 {
		slog.Error("Error: channels must be ≥ 1")
		os.Exit(1)
	}

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	var wg sync.WaitGroup
	for i := 0; i < *count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			runSubscriber(idx, *baseURL, *stream)
		}(i)
	}
	wg.Wait()
	slog.Info("All subscribers completed")

}

func runSubscriber(idx int, baseURL, stream string) {
	// Each subscriber reads from its own stream: e.g. stream_0, stream_1, …
	url := fmt.Sprintf("%s/%s_%d", baseURL, stream, idx)
	sub := trickle.NewTrickleSubscriber(url)

	for {
		resp, err := sub.Read()
		if err != nil {
			if errors.Is(err, trickle.EOS) {
				slog.Info("End-of-stream signal received", "url", url)
				return
			}
			if errors.Is(err, trickle.StreamNotFoundErr) {
				slog.Info("Stream not found", "url", url)
				return
			}
			var sequenceNonexistent *trickle.SequenceNonexistent
			if errors.As(err, &sequenceNonexistent) {
				// stream exists but segment doesn't, so skip to leading edge
				slog.Warn("Segment doesn't exist", "url", url, "seq", sequenceNonexistent.Seq, "latest", sequenceNonexistent.Latest)
				sub.SetSeq(sequenceNonexistent.Latest)
				time.Sleep(10 * time.Millisecond)
				continue
			}
			slog.Error("Error reading segment", "url", url, "err", err)
			return
		}

		// Compute SHA-256 while consuming the body
		hasher := sha256.New()
		_, err = io.Copy(hasher, resp.Body)
		resp.Body.Close()
		seq := trickle.GetSeq(resp)
		if err != nil {
			slog.Error("Failed while reading segment", "seq", seq, "err", err)
			return
		}

		digest := hex.EncodeToString(hasher.Sum(nil))
		slog.Info(fmt.Sprintf("%02d-%04d", idx, seq), "SHA-256", digest)
	}
}
