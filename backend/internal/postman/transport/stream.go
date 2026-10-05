package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"lunar/backend/internal/postman/application"
	"lunar/backend/internal/postman/domain"
)

const (
	stageEventName  = "stage"
	resultEventName = "result"
	errorEventName  = "error"
)

type errorPayload struct {
	Error string `json:"error"`
}

type Stream struct {
	writer    http.ResponseWriter
	flusher   http.Flusher
	isStarted bool
}

func NewStream(w http.ResponseWriter) (*Stream, error) {
	flusher, isFlushable := w.(http.Flusher)
	if !isFlushable {
		return nil, errors.New("the response writer does not support streaming")
	}
	return &Stream{writer: w, flusher: flusher}, nil
}

func (s *Stream) IsStarted() bool {
	return s.isStarted
}

func (s *Stream) Stage(event domain.StageEvent) error {
	s.start()
	return s.writeEvent(stageEventName, event)
}

func (s *Stream) Result(result application.Result) error {
	s.start()
	return s.writeEvent(resultEventName, result)
}

func (s *Stream) Error(message string) error {
	s.start()
	return s.writeEvent(errorEventName, errorPayload{Error: message})
}

func (s *Stream) start() {
	if s.isStarted {
		return
	}
	s.isStarted = true
	s.writer.Header().Set("Content-Type", "text/event-stream")
	s.writer.Header().Set("Cache-Control", "no-cache")
	s.writer.Header().Set("Connection", "keep-alive")
	s.writer.Header().Set("X-Accel-Buffering", "no")
	s.writer.WriteHeader(http.StatusOK)
	s.flusher.Flush()
}

func (s *Stream) writeEvent(name string, payload any) error {
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
