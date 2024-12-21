package main

import (
	"flag"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
	"trickle"
)

// Listens to new streams from MediaMTX and publishes
// to trickle HTTP server under the same name
// then concurrently subscribes and writes to an outfile

var (
	baseURL *string
	outFile *string
)

type SegmentPoster struct {
	tricklePublisher *trickle.TricklePublisher
}

func (sp *SegmentPoster) NewSegment(reader trickle.CloneableReader) {
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

func runSubscribe(streamName string) error {
	client := trickle.NewTrickleSubscriber(*baseURL + "/" + streamName)
	outPipe, err := os.OpenFile(*outFile, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("Error opening subscribe output", "stream", streamName, "err", err)
		return err
	}
	defer outPipe.Close()
	slog.Info("Subscribing", "stream", streamName)
	for {
		resp, err := client.Read()
		if err != nil {
			slog.Error("Error getting client reader", "stream", streamName, "err", err)
			break
		}
		idx := trickle.GetSeq(resp)
		n, err := io.Copy(outPipe, resp.Body)
		if err != nil {
			slog.Error("Error copying to output", "stream", streamName, "idx", idx, "err", err, "copied", n)
			break
		}
		resp.Body.Close()
	}
	slog.Info("Subscription stopped", "stream", streamName)
	return nil
}

func newPublish(w http.ResponseWriter, r *http.Request) {
	streamName := r.PathValue("streamName")

	slog.Info("Starting stream", "streamName", streamName)

	go func() {
		sp := segmentPoster(streamName)
		defer sp.tricklePublisher.Close()
		go func() {
			// give it some time for the publisher to latch on
			time.Sleep(2 * time.Second)
			runSubscribe(streamName)
		}()
		(&trickle.MediaSegmenter{}).RunSegmentation("rtmp://localhost/"+streamName, sp.NewSegment)
	}()
}

func listen(host string) {
	srv := &http.Server{
		Addr: host,
	}
	http.HandleFunc("POST /{streamName}/{$}", newPublish)
	slog.Info("Listening for MediaMTX", "host", host)
	log.Fatal(srv.ListenAndServe())
}

func main() {

	// Check some command-line arguments
	baseURL = flag.String("url", "http://localhost:2939", "Base URL for the stream")
	outFile = flag.String("out", "", "Output file name (required)")
	flag.Parse()
	if *outFile == "" {
		log.Fatalf("Error: Output file is required. Use -out flag to specify the output file.")
	}

	listen(":2938")
}
