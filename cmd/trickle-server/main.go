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
	p := flag.String("path", "/", "URL to publish streams to")
	addr := flag.String("addr", ":2939", "Address to bind to")
	flag.Parse()

	srv := &http.Server{
		// say max segment size is 20 secs
		// we can allow 2 * 20 secs given preconnects
		Addr:         *addr,
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 45 * time.Second,
	}

	trickleSrv := trickle.ConfigureServer(trickle.TrickleServerConfig{
		BasePath:   EnsureSlash(*p),
		Changefeed: true,
		Autocreate: true,
	})
	changefeedSubscribe(trickleSrv)
	log.Println("Server started at " + *addr)
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
