package task

// ManagerConfig holds the configuration needed to create a Manager.
type ManagerConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Concurrency   int
	Queues        []string
	// ErrorHandler is invoked when a task ultimately fails. If nil, errors are
	// printed to stdout. Inject app.Log here to route failures into zap.
	ErrorHandler func(taskType string, err error)
}

// Manager holds the task client, worker, and scheduler instances.
type Manager struct {
	Client    *Client
	Worker    *Worker
	Scheduler *Scheduler
}

func NewManager(cfg ManagerConfig) *Manager {
	queues := make(map[string]int)
	if len(cfg.Queues) == 0 {
		cfg.Queues = []string{"default"}
	}
	for i, q := range cfg.Queues {
		queues[q] = len(cfg.Queues) - i
	}

	return &Manager{
		Client:    NewClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB),
		Worker:    NewWorker(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.Concurrency, queues, cfg.ErrorHandler),
		Scheduler: NewScheduler(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB),
	}
}

func (m *Manager) Close() {
	if m.Client != nil {
		_ = m.Client.Close()
	}
}
