package trickle

import (
	"sync"
)

type SegmentHandler func(reader CloneableReader)

func NoopReader(reader CloneableReader) {
	// don't have to do anything here
}

type SwitchableSegmentReader struct {
	mu     sync.RWMutex
	reader SegmentHandler
}

func NewSwitchableSegmentReader() *SwitchableSegmentReader {
	return &SwitchableSegmentReader{
		reader: NoopReader,
	}
}

func (sr *SwitchableSegmentReader) SwitchReader(newReader SegmentHandler) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.reader = newReader
}

func (sr *SwitchableSegmentReader) Read(reader CloneableReader) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	sr.reader(reader)
}
