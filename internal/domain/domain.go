package domain

import "context"

type Task struct {
	ID      string `json:"id,omitempty"`
	Date    string `json:"date,omitempty"`
	Title   string `json:"title,omitempty"`
	Comment string `json:"comment,omitempty"`
	Repeat  string `json:"repeat,omitempty"`
}

type TaskInput struct {
	ID      *string `json:"id,omitempty"`
	Date    *string `json:"date,omitempty"`
	Title   *string `json:"title,omitempty"`
	Comment *string `json:"comment,omitempty"`
	Repeat  *string `json:"repeat,omitempty"`
}

type Filter struct {
	ID         *int
	SearchTerm string
	Date       string
	Limit      int
}

type TaskRepository interface {
	FindTask(ctx context.Context, filter *Filter) ([]*Task, error)
	CreateTask(ctx context.Context, task *Task) (int64, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, id *int) error
	Close()
}

type TaskOption func(*Task)

func NewTask(opts ...TaskOption) *Task {
	t := &Task{}

	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (r *TaskInput) TaskToOptions() []TaskOption {
	var opts []TaskOption
	if r.ID != nil {
		opts = append(opts, WithID(*r.ID))
	}
	if r.Date != nil {
		opts = append(opts, WithDate(*r.Date))
	}
	if r.Title != nil {
		opts = append(opts, WithTitle(*r.Title))
	}
	if r.Comment != nil {
		opts = append(opts, WithComment(*r.Comment))
	}
	if r.Repeat != nil {
		opts = append(opts, WithRepeat(*r.Repeat))
	}
	return opts
}

func WithID(id string) TaskOption {
	return func(task *Task) {
		task.ID = id
	}
}

func WithDate(date string) TaskOption {
	return func(task *Task) {
		task.Date = date
	}
}

func WithTitle(title string) TaskOption {
	return func(task *Task) {
		task.Title = title
	}
}

func WithComment(comment string) TaskOption {
	return func(task *Task) {
		task.Comment = comment
	}
}

func WithRepeat(repeat string) TaskOption {
	return func(task *Task) {
		task.Repeat = repeat
	}
}
