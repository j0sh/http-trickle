package main

import (
	"errors"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
	"trickle"
)

// Listens to new streams from MediaMTX and publishes
// to trickle HTTP server under the same name

var baseURL *url.URL

type SegmentPoster struct {
	tricklePublisher *trickle.TricklePublisher
	checker          *SlowServerChecker
}

func (sp *SegmentPoster) NewSegment(reader trickle.CloneableReader) {
	kount, atMax := sp.checker.BeginSegment()
	if atMax {
		slog.Info("Server is too slow!")
		return
	}
	slog.Info("Starting", "seq", kount)
	go func(seq int) {
		defer sp.checker.EndSegment()
		writer, err := sp.tricklePublisher.Next()
		if err != nil {
			slog.Info("weird error not sure why")
			return
		}
		for {
			currentSeq := sp.checker.GetCount()
			if seq != currentSeq {
				slog.Info("Next segment has already started; skipping this one seq=%d currentSeq=%d", seq, currentSeq)
				return
			}
			n, err := writer.Write(reader)
			if errors.Is(err, trickle.StreamNotFoundErr) {
				slog.Info("no stream found, exiting")
				return
			} else if err != nil {
				// lets retry
				if n > 0 {
					slog.Info("Error publishing segment; dropping remainder", "wrote", n, "err", err)
					return
				}
				reader = reader.Clone()
				time.Sleep(100 * time.Millisecond)
				slog.Info("Retrying segment", "err", err)
			} else {
				// no error, so exit
				break
			}
		}
	}(kount)
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
		checker:          &SlowServerChecker{},
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

// Detect 'slow' orchs by keeping track of in-flight segments
// Count the difference between segments produced and segments completed
type SlowServerChecker struct {
	mu            sync.Mutex
	segmentCount  int
	completeCount int
}

// Number of in flight segments to allow.
// Should generally not be less than 1, because
// sometimes the beginning of the current segment
// may briefly overlap with the end of the previous segment
const maxInflightSegments = 3

// Returns the number of segments begun so far and
// whether the max number of inflight segments was hit.
// Number of segments is not incremented if inflight max is hit.
// If inflight max is hit, returns true, false otherwise.
func (s *SlowServerChecker) BeginSegment() (int, bool) {
	// Returns `false` if there are multiple segments in-flight
	// this means the server is slow reading them
	// If all-OK, returns `true`
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.segmentCount >= s.completeCount+maxInflightSegments {
		// There is > 1 segment in flight ... server is slow reading
		return s.segmentCount, true
	}
	s.segmentCount += 1
	return s.segmentCount, false

}

func (s *SlowServerChecker) EndSegment() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completeCount += 1
}

func (s *SlowServerChecker) GetCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.segmentCount
}
func main() {
	handleArgs()
	listen(":2938")
}
