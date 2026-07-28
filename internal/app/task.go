package app

import "github.com/kar1hsu/backplane/internal/pkg/task"

var TaskMgr *task.Manager

func InitTask() {
	TaskMgr = task.NewManager(task.ManagerConfig{
		RedisAddr:     Cfg.Redis.Addr(),
		RedisPassword: Cfg.Redis.Password,
		RedisDB:       Cfg.Redis.DB,
		Concurrency:   Cfg.Task.Concurrency,
		Queues:        Cfg.Task.Queues,
		ErrorHandler: func(taskType string, err error) {
			Log.Errorw("task failed", "type", taskType, "err", err)
		},
	})
	Log.Info("task manager initialized")
}
