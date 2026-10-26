package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type EventStream struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	started bool
}

func NewEventStream(w http.ResponseWriter) (*EventStream, error) {
	flusher, isFlushable := w.(http.Flusher)
	if !isFlushable {
		return nil, errors.New("the response writer does not support streaming")
	}
	return &EventStream{writer: w, flusher: flusher}, nil
}

func (s *EventStream) IsStarted() bool {
	return s.started
}

func (s *EventStream) Phase(phase string) error {
	return s.writeEvent("phase", map[string]string{"phase": phase})
}

func (s *EventStream) Output(chunk string) error {
	return s.writeEvent("output", map[string]string{"chunk": chunk})
}

func (s *EventStream) Done(payload any) error {
	return s.writeEvent("done", payload)
}

func (s *EventStream) Error(message string) error {
	return s.writeEvent("error", map[string]string{"error": message})
}

func (s *EventStream) writeEvent(name string, payload any) error {
	s.start()

	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s event: %w", name, err)
	}
	if _, err := fmt.Fprintf(s.writer, "event: %s\ndata: %s\n\n", name, encoded); err != nil {
		return fmt.Errorf("write %s event: %w", name, err)
	}
	s.flusher.Flush()
	return nil
}

func (s *EventStream) start() {
	if s.started {
		return
	}
	s.started = true
	s.writer.Header().Set("Content-Type", "text/event-stream")
	s.writer.Header().Set("Cache-Control", "no-cache")
	s.writer.Header().Set("Connection", "keep-alive")
	s.writer.Header().Set("X-Accel-Buffering", "no")
	s.writer.WriteHeader(http.StatusOK)
	s.flusher.Flush()
}
