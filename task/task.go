package task

type Task struct {
	Id      int
	Execute func() error
	Payload any
	Status  TaskStatus
}

type TaskStatus int

const (
	Pending TaskStatus = iota
	Running
	Completed
	Failed
)
