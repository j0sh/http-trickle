package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Entry struct {
	Timestamp time.Time
	SHA       string
}

var lineRegex = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d+).*(\d{2}-\d{4}).*SHA-256=([0-9a-f]+)`)

// parseLog reads a log file and returns a map from ID to Entry
func parseLog(path string) (map[string]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := make(map[string]Entry)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := lineRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		tsStr := matches[1]
		id := matches[2]
		sha := matches[3]

		// Try parsing microseconds, then milliseconds
		ts, err := time.Parse("2006/01/02 15:04:05.000000", tsStr)
		if err != nil {
			ts, err = time.Parse("2006/01/02 15:04:05.000", tsStr)
			if err != nil {
				continue
			}
		}

		entries[id] = Entry{Timestamp: ts, SHA: sha}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func summaryStats(data []float64) (mean, median, p90, p99 float64) {
	n := len(data)
	if n == 0 {
		return 0, 0, 0, 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean = sum / float64(n)

	sort.Float64s(data)
	// median
	if n%2 == 1 {
		median = data[n/2]
	} else {
		median = (data[n/2-1] + data[n/2]) / 2
	}
	// percentiles
	idx90 := int(float64(n)*0.9+0.5) - 1
	if idx90 < 0 {
		idx90 = 0
	} else if idx90 >= n {
		idx90 = n - 1
	}
	idx99 := int(float64(n)*0.99+0.5) - 1
	if idx99 < 0 {
		idx99 = 0
	} else if idx99 >= n {
		idx99 = n - 1
	}
	p90 = data[idx90]
	p99 = data[idx99]
	return
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <subscriber_logs> <publisher_logs>\n", os.Args[0])
		os.Exit(1)
	}

	subscriberPath := os.Args[1]
	publisherPath := os.Args[2]

	subscriberLogs, err := parseLog(subscriberPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing subscriber logs (%s): %v\n", subscriberPath, err)
		os.Exit(1)
	}
	publisherLogs, err := parseLog(publisherPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing publisher logs (%s): %v\n", publisherPath, err)
		os.Exit(1)
	}

	// IDs in publisher logs but not in subscriber logs
	var onlyInPublisher []string
	for id := range publisherLogs {
		if _, ok := subscriberLogs[id]; !ok {
			onlyInPublisher = append(onlyInPublisher, id)
		}
	}
	sort.Strings(onlyInPublisher)
	fmt.Printf("Entries in publisher logs but not in subscriber logs: %d\n", len(onlyInPublisher))
	if len(onlyInPublisher) > 0 {
		fmt.Printf("  %s\n\n", strings.Join(onlyInPublisher, ", "))
	} else {
		fmt.Println()
	}

	// Compute deltas (ms) and SHA match for common IDs
	type Record struct {
		Seq     string
		DeltaMs float64
		SubSHA  string
		PubSHA  string
		Match   bool
	}
	byIdx := make(map[string][]Record)

	for id, pubEntry := range publisherLogs {
		subEntry, ok := subscriberLogs[id]
		if !ok {
			continue
		}
		parts := strings.SplitN(id, "-", 2)
		idx, seq := parts[0], parts[1]
		deltaMs := pubEntry.Timestamp.Sub(subEntry.Timestamp).Seconds() * 1000
		match := subEntry.SHA == pubEntry.SHA
		byIdx[idx] = append(byIdx[idx], Record{Seq: seq, DeltaMs: deltaMs, SubSHA: subEntry.SHA, PubSHA: pubEntry.SHA, Match: match})
	}

	// Print summary for each idx
	var idxs []string
	for idx := range byIdx {
		idxs = append(idxs, idx)
	}
	sort.Strings(idxs)

	for _, idx := range idxs {
		recs := byIdx[idx]
		// separate matched deltas and mismatches
		var deltas []float64
		var mismatches []Record
		for _, r := range recs {
			if r.Match {
				deltas = append(deltas, r.DeltaMs)
			} else {
				mismatches = append(mismatches, r)
			}
		}

		mean, median, p90, p99 := summaryStats(deltas)
		fmt.Printf("idx %s: count=%d, mean=%.3f ms, median=%.3f ms, p90=%.3f ms, p99=%.3f ms\n", idx, len(deltas), mean, median, p90, p99)
		if len(mismatches) > 0 {
			fmt.Println("Mismatched SHAs:")
			sort.Slice(mismatches, func(i, j int) bool { return mismatches[i].Seq < mismatches[j].Seq })
			for _, r := range mismatches {
				fmt.Printf("  seq %s: subscriber=%s, publisher=%s\n", r.Seq, r.SubSHA, r.PubSHA)
			}
		}
		fmt.Println()
	}
}
