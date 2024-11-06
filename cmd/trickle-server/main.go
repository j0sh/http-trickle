package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"trickle"
)

func main() {
	srv := &http.Server{
		// say max segment size is 20 secs
		// we can allow 2 * 20 secs given preconnects
		Addr:         ":2939",
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 45 * time.Second,
	}

	p := flag.String("path", "/", "URL to publish streams to")
	flag.Parse()
	trickleSrv := trickle.ConfigureServer(trickle.TrickleServerConfig{
		BasePath:   EnsureSlash(*p),
		Changefeed: true,
	})
	changefeedSubscribe(trickleSrv)
	log.Println("Server started at :2939")
	log.Fatal(srv.ListenAndServe())
}

func changefeedSubscribe(srv *trickle.Server) {
	go func() {
		sub := trickle.NewLocalSubscriber(srv, trickle.CHANGEFEED)
		for true {
			part, err := sub.Read()
			if err != nil {
				log.Fatal("Changefeed error", err)
			}
			b, err := io.ReadAll(part.Reader)
			if err != nil {
				log.Fatal("Changefeed read error", err)
			}
			log.Println("Changefeed", string(b))
		}
	}()
}

func EnsureSlash(s string) string {
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	if !strings.HasSuffix(s, "/") {
		s = s + "/"
	}
	return s
}
