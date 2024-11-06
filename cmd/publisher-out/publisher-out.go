package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
	"trickle"
)

// Listens to new streams from MediaMTX and publishes
// to trickle HTTP server under the same name
// Also subscribes to channels with an `-out` prefix
// and pushes those out streams into MediaMTX

var baseURL *url.URL

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
	defer r.Body.Close()

	if strings.HasSuffix(streamName, "-out") {
		// don't subscribe to `-out` since we will be pushing that
		return
	}

	slog.Info("Publishing stream", "streamName", streamName)

	go func() {
		sp := segmentPoster(streamName)
		defer sp.tricklePublisher.Close()
		ms := &trickle.MediaSegmenter{}
		ms.RunSegmentation("rtmp://localhost/"+streamName, sp.NewSegment)
		slog.Info("Unpublishing stream", "streamName", streamName)
	}()
}

func runSubscribe(streamName string) error {
	sub := trickle.NewTrickleSubscriber(baseURL.String() + streamName)
	r, w, err := os.Pipe()
	if err != nil {
		slog.Error("Could not open ffmpeg pipe", "stream", streamName, "err", err)
		return err
	}
	slog.Info("Subscribing", "stream", streamName)
	ffmpegDone := make(chan struct{})
	subscriptionDone := make(chan struct{})
	rc := retryCounter{}

	go func() {
		// lpms currently does not work on joshs local mac
		defer close(ffmpegDone)
		ffmpegPath := os.Getenv("FFMPEG_PATH")
		if ffmpegPath == "" {
			ffmpegPath = "ffmpeg"
		}
		for {
			cmd := exec.Command(ffmpegPath, "-i", "-", "-c", "copy", "-f", "flv", "rtmp://localhost/"+streamName)
			cmd.Stdin = r
			out, err := cmd.CombinedOutput()
			if err != nil {
				slog.Error("Error running ffmpeg", "err", err)
			}
			slog.Debug(string(out))
			select {
			case <-subscriptionDone:
				return
			default:
				// probably premature exit so try again
				if !rc.checkRetry() {
					slog.Info("Max retries hit, closing", "stream", streamName)
					return
				} else {
					slog.Info("Retrying ffmpeg input", "stream", streamName)
				}
			}
		}
	}()

	for {
		resp, err := sub.Read()
		if err != nil {
			if errors.Is(err, trickle.EOS) {
				slog.Info("End of stream signal", "stream", streamName)
			}
			slog.Error("Error reading subscription", "stream", streamName, "err", err)
			break
		}
		if resp.StatusCode == http.StatusOK {
			io.Copy(w, resp.Body)
			resp.Body.Close()
		} else {
			slog.Error("Non-200 status code", "stream", streamName, "code", resp.StatusCode)
			resp.Body.Close()
			break
		}
	}
	close(subscriptionDone)
	w.Close()
	<-ffmpegDone
	slog.Info("Subscription stopped", "stream", streamName)
	return nil
}

func changefeedSubscribe() {
	go func() {
		client := trickle.NewTrickleSubscriber(baseURL.String() + trickle.CHANGEFEED)
		for {
			res, err := client.Read()
			if err != nil {
				log.Fatal("Failed to read changefeed:", err)
				continue
			}
			ch := trickle.Changefeed{}
			if err := json.NewDecoder(res.Body).Decode(&ch); err != nil {
				slog.Error("Failed to deserialize changefeed", "seq", trickle.GetSeq(res), "err", err)
				continue
			}
			slog.Info("Changefeed received", "ch", ch)
			// Subscribe to new streams with the sufix "-out"
			for _, stream := range ch.Added {
				if strings.HasSuffix(stream, "-out") {
					go runSubscribe(stream)
				}
			}
		}
	}()
}

type retryCounter struct {
	retryCount int
	firstRetry time.Time
}

func (rc *retryCounter) checkRetry() bool {
	const maxRetries = 5
	const retryInterval = 30 * time.Second

	// Check if the retry limit has been exceeded
	if rc.retryCount == 0 {
		rc.firstRetry = time.Now()
	}
	if time.Since(rc.firstRetry) > retryInterval {
		// Reset counter and timestamp after interval
		rc.retryCount = 0
		rc.firstRetry = time.Now()
	}

	rc.retryCount++
	if rc.retryCount > maxRetries {
		return false
	}
	return true
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
	//slog.SetLogLoggerLevel(slog.LevelDebug)
	handleArgs()
	changefeedSubscribe()
	listen(":2938")
}
