package task

type Task struct {
	Id         int
	Execute    func(args ...any) error
	Type       string
	Payload    string
	Status     TaskStatus
	RetryCount int
	MaxRetries int
}

type TaskStatus int

const (
	Pending TaskStatus = iota
	Running
	Completed
	Failed
)

func (s TaskStatus) String() string {
	switch s {
	case Pending:
		return "pending"
	case Running:
		return "running"
	case Completed:
		return "completed"
	case Failed:
		return "failed"
	default:
		return "unknown"
	}
}
