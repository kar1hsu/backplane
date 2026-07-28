package tasks

import (
	"time"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/pkg/task"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
)

// RegisterHandlers registers all task handlers on the worker.
func RegisterHandlers(w *task.Worker) {
	w.Handle(TypeEmailSend, HandleEmailSend)
	w.Handle(TypeCleanup, HandleCleanup)
}

// RegisterCronJobs registers all scheduled tasks on the scheduler.
func RegisterCronJobs(s *task.Scheduler) {
	jobs := []task.CronTask{
		// Daily cleanup at 2:00 AM. Unique TTL (< 24h) dedupes enqueues so an
		// accidentally started second scheduler instance won't run cleanup twice.
		{
			Cron:     utils.DailyAt(2, 0),
			TypeName: TypeCleanup,
			Payload:  nil,
			Queue:    "low",
			Unique:   23 * time.Hour,
		},
	}
	for _, job := range jobs {
		if _, err := s.Register(job); err != nil {
			app.Log.Errorf("register cron job [%s] failed: %v", job.TypeName, err)
		}
	}
}
