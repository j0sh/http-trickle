package trickle

import (
	"bytes"
	"io"
	"log/slog"
	"sync"
)

type CloneableReader interface {
	io.Reader
	Clone() CloneableReader
}

type TrickleWriter struct {
	mu     *sync.Mutex
	cond   *sync.Cond
	buffer *bytes.Buffer
	closed bool
}

type TrickleReader struct {
	source  *TrickleWriter
	readPos int
}

func NewTrickleWriter() *TrickleWriter {
	mu := &sync.Mutex{}
	return &TrickleWriter{
		buffer: new(bytes.Buffer),
		cond:   sync.NewCond(mu),
		mu:     mu,
	}
}

func (tw *TrickleWriter) Write(data []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	// Write to buffer
	n, err := tw.buffer.Write(data)

	// Signal waiting readers
	tw.cond.Broadcast()

	return n, err
}

func (s *TrickleWriter) readData(startPos int) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		totalLen := s.buffer.Len()
		if startPos < totalLen {
			data := s.buffer.Bytes()[startPos:totalLen]
			return data, s.closed
		}
		if startPos > totalLen {
			slog.Info("Invalid start pos, invoking eof")
			return nil, true
		}
		if s.closed {
			return nil, true
		}
		// Wait for new data
		s.cond.Wait()
	}
}

func (tw *TrickleWriter) Close() {
	if tw == nil {
		return // sometimes happens, weird
	}
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.closed {
		tw.closed = true
		tw.cond.Broadcast()
	}
}

// Creates a buffered reader that does *not* block the writer
func (tw *TrickleWriter) MakeReader() CloneableReader {
	return &TrickleReader{
		source: tw,
	}
}

func (tr *TrickleReader) Read(p []byte) (int, error) {
	data, eof := tr.source.readData(tr.readPos)
	toRead := len(p)
	if len(data) < toRead {
		toRead = len(data)
	}

	copy(p, data[:toRead])
	tr.readPos += toRead

	var err error = nil
	if eof {
		err = io.EOF
	}

	return toRead, err
}

func (tr *TrickleReader) Clone() CloneableReader {
	return &TrickleReader{
		source: tr.source,
	}
}
