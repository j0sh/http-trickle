package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
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

	slog.Info("Starting stream", "streamName", streamName)

	go func() {
		sp := segmentPoster(streamName)
		defer sp.tricklePublisher.Close()
		ms := &trickle.MediaSegmenter{}
		ms.RunSegmentation("rtmp://localhost/"+streamName, sp.NewSegment)
		slog.Info("Closing stream", "streamName", streamName)
	}()
}

func runSubscribe(streamName string) error {
	sub := trickle.NewTrickleSubscriber(baseURL.String() + streamName)
	r, w, err := os.Pipe()
	if err != nil {
		slog.Error("Could not open ffmpeg pipe", "stream", streamName, "err", err)
		return err
	}
	defer w.Close()
	slog.Info("Subscribing", "stream", streamName)

	go func() {
		// lpms currently does not work on joshs local mac
		ffmpegPath := os.Getenv("FFMPEG_PATH")
		if ffmpegPath == "" {
			ffmpegPath = "ffmpeg"
		}
		cmd := exec.Command(ffmpegPath, "-i", "-", "-c", "copy", "-f", "flv", "rtmp://localhost/"+streamName)
		cmd.Stdin = r
		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("Error running ffmpeg", "err", err)
		}
		fmt.Println(string(out))
	}()

	for {
		resp, err := sub.Read()
		if err != nil {
			slog.Error("Error reading subscription", "stream", streamName, "err", err)
			break
		}
		io.Copy(w, resp.Body)
		resp.Body.Close()
	}
	slog.Info("Subscription stopped", "stream", streamName)
	return nil
}

func changefeedSubscribe() {
	go func() {
		client := trickle.NewTrickleSubscriber(baseURL.String() + trickle.CHANGEFEED)
		for i := 0; true; i++ {
			res, err := client.Read()
			if err != nil {
				log.Fatal("Failed to read changefeed:", err)
				continue
			}
			ch := trickle.Changefeed{}
			if err := json.NewDecoder(res.Body).Decode(&ch); err != nil {
				slog.Error("Failed to deserialize changefeed", "err", err)
			}
			slog.Info("Changefeed received", "ch", ch)
			// Subscribe to new streams with the sufix "-out"
			for _, stream := range ch.Added {
				if strings.HasSuffix(stream, "-out") {
					go func() {
						runSubscribe(stream)
					}()
				}
			}
		}
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
	changefeedSubscribe()
	listen(":2938")
}
