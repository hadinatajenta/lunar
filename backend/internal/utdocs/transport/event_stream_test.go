package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type nonFlushableWriter struct {
	header http.Header
}

func (w *nonFlushableWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *nonFlushableWriter) Write(_ []byte) (int, error) {
	return 0, nil
}

func (w *nonFlushableWriter) WriteHeader(_ int) {}

func TestEventStreamWritesTheProtocolFrames(t *testing.T) {
	recorder := httptest.NewRecorder()
	stream, err := NewEventStream(recorder)
	if err != nil {
		t.Fatalf("expected a stream, got %v", err)
	}
	if stream.IsStarted() {
		t.Fatal("expected the stream not to have started")
	}

	if err := stream.Phase("scan"); err != nil {
		t.Fatalf("cannot write the phase event: %v", err)
	}
	if !stream.IsStarted() {
		t.Fatal("expected the stream to have started")
	}
	if err := stream.Output("ok\n"); err != nil {
		t.Fatalf("cannot write the output event: %v", err)
	}
	if err := stream.Done(map[string]string{"run_id": "run-1"}); err != nil {
		t.Fatalf("cannot write the done event: %v", err)
	}
	if err := stream.Error("failed"); err != nil {
		t.Fatalf("cannot write the error event: %v", err)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}
	body := recorder.Body.String()
	for _, frame := range []string{
		"event: phase\ndata: {\"phase\":\"scan\"}\n\n",
		"event: output\ndata: {\"chunk\":\"ok\\n\"}\n\n",
		"event: done\ndata: {\"run_id\":\"run-1\"}\n\n",
		"event: error\ndata: {\"error\":\"failed\"}\n\n",
	} {
		if !strings.Contains(body, frame) {
			t.Fatalf("expected the stream to contain %q, got %s", frame, body)
		}
	}
	if !recorder.Flushed {
		t.Fatal("expected the stream to flush")
	}
}

func TestEventStreamRequiresAFlushableWriter(t *testing.T) {
	if _, err := NewEventStream(&nonFlushableWriter{}); err == nil {
		t.Fatal("expected a non flushable writer to be rejected")
	}
}
