package main

import (
	"flag"
	"io"
	"log"
	"os"

	"trickle"
)

// TrickleReader example usage
func main() {

	// Check some command-line arguments
	baseURL := flag.String("url", "http://localhost:2939", "Base URL for the stream")
	streamName := flag.String("stream", "", "Stream name (required)")
	flag.Parse()
	if *streamName == "" {
		log.Fatalf("Error: stream name is required. Use -stream flag to specify the stream name.")
	}

	client := trickle.NewTrickleSubscriber(*baseURL + "/" + *streamName)

	for i := 0; true; i++ {
		// Read and process the first segment
		resp, err := client.Read()
		idx := trickle.GetSeq(resp)
		if err != nil {
			log.Fatal("Failed to read segment", idx, err)
			continue
		}
		n, err := io.Copy(os.Stdout, resp.Body)
		if err != nil {
			log.Fatal("Failed to record segment", idx, err)
			continue
		}
		resp.Body.Close()
		log.Println("--- End of Segment ", idx, "part", i, "bytes", trickle.HumanBytes(n), " ---")
	}
	log.Println("Completing", *streamName)
}
