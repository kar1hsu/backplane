package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kar1hsu/backplane/internal/app"
)

type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func HandleEmailSend(ctx context.Context, payload []byte) error {
	var p EmailPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("unmarshal email payload: %w", err)
	}

	// TODO: replace with real email sending logic
	app.Log.Infof("[task] sending email to=%s subject=%s", p.To, p.Subject)
	return nil
}
