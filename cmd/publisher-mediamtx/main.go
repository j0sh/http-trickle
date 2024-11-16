package main

import (
	"flag"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"trickle"
)

// Listens to new streams from MediaMTX and publishes
// to trickle HTTP server under the same name

var baseURL *url.URL

type SegmentPoster struct {
	tricklePublisher *trickle.TricklePublisher
}

func (sp *SegmentPoster) NewSegment(reader io.Reader) {
	go func() {
		// NB: This blocks! Very bad!
		err := sp.tricklePublisher.Write(reader)
		if err == trickle.StreamNotFoundErr {
			slog.Info("no stream found, exiting")
		}
		if err != nil {
			// discard anything that remains from the reader
			io.Copy(io.Discard, reader)
		} else {
		}
	}()
}

func segmentPoster(streamName string) *SegmentPoster {
	u, err := url.JoinPath(baseURL.String(), streamName)
	if err != nil {
		panic(err)
	}
	c, err := trickle.NewTricklePublisher(u)
	if err != nil {
		panic(err)
	}
	return &SegmentPoster{
		tricklePublisher: c,
	}
}

func listen(host string) {
	srv := &http.Server{
		Addr: host,
	}
	http.HandleFunc("POST /{streamName}/{$}", newPublish)
	slog.Info("Listening for MediaMTX", "host", host)
	log.Fatal(srv.ListenAndServe())
}

func newPublish(w http.ResponseWriter, r *http.Request) {
	streamName := r.PathValue("streamName")

	slog.Info("Starting stream", "streamName", streamName)

	go func() {
		sp := segmentPoster(streamName)
		defer sp.tricklePublisher.Close()
		ms := &trickle.MediaSegmenter{}
		ms.RunSegmentation("rtmp://localhost/"+streamName, sp.NewSegment)
		slog.Info("Closing stream", "streamName", streamName)
	}()
}

func handleArgs() {
	u := flag.String("url", "http://localhost:2939/", "URL to publish streams to")
	flag.Parse()
	var err error
	baseURL, err = url.Parse(*u)
	if err != nil {
		log.Fatal(err)
	}
	parsedURL := *baseURL
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		log.Fatal("Invalid URL scheme ", parsedURL.Scheme, " only http and https are allowed.", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		log.Fatal("Missing host for URL")
	}
	slog.Info("Trickle server", "url", parsedURL.String())
}

func main() {
	handleArgs()
	listen(":2938")
}
