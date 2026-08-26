package task

import (
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
)

func TestClientEnqueueToQueue(t *testing.T) {
	server := miniredis.RunT(t)
	client := NewClient(server.Addr(), "", 0)
	t.Cleanup(func() { _ = client.Close() })

	type payload struct {
		UserID uint   `json:"user_id"`
		Email  string `json:"email"`
	}
	wantPayload := payload{UserID: 42, Email: "user@example.com"}
	info, err := client.EnqueueToQueue("email:send", wantPayload, "critical")
	if err != nil {
		t.Fatalf("EnqueueToQueue() error = %v", err)
	}
	if info.Type != "email:send" || info.Queue != "critical" {
		t.Fatalf("task info = %+v", info)
	}

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: server.Addr()})
	t.Cleanup(func() { _ = inspector.Close() })
	pending, err := inspector.ListPendingTasks("critical")
	if err != nil {
		t.Fatalf("ListPendingTasks() error = %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending tasks = %d, want 1", len(pending))
	}
	if pending[0].Type != "email:send" {
		t.Fatalf("task type = %q", pending[0].Type)
	}
	var gotPayload payload
	if err := json.Unmarshal(pending[0].Payload, &gotPayload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if gotPayload != wantPayload {
		t.Fatalf("payload = %+v, want %+v", gotPayload, wantPayload)
	}
}

func TestClientRejectsUnserializablePayload(t *testing.T) {
	server := miniredis.RunT(t)
	client := NewClient(server.Addr(), "", 0)
	t.Cleanup(func() { _ = client.Close() })

	if _, err := client.Enqueue("invalid:payload", make(chan int)); err == nil {
		t.Fatal("Enqueue() accepted an unserializable payload")
	}
}
