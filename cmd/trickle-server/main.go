package main

import (
	"flag"
	"log"
	"net/http"
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
	trickle.ConfigureServer(trickle.TrickleServerConfig{
		BasePath: *p,
	})
	log.Println("Server started at :2939")
	log.Fatal(srv.ListenAndServe())
}
