package dccompute

import (
	"fmt"
	"time"

	"github.com/satori/go.uuid"
)

// Task statuses
const (
	TaskStatusTodo    = "todo"
	TaskStatusWaiting = "waiting"
	TaskStatusPending = "pending"
	TaskStatusDone    = "done"
	TaskStatusFailed  = "failed"
)

var (
	// ValidStatuses is a set of all possible values for the "status" field
	ValidStatuses = map[string]struct{}{
		TaskStatusTodo:    struct{}{},
		TaskStatusWaiting: struct{}{},
		TaskStatusPending: struct{}{},
		TaskStatusDone:    struct{}{},
		TaskStatusFailed:  struct{}{},
	}
)

// Checkable is an Interface for things that can be Checked (i.e. validated after a JSON parsing for
// instance)
type Checkable interface {
	Check() (err error)
}

// Preduplet describes a prediction task.
type Preduplet struct {
	Checkable

	ID             uuid.UUID   `json:"uuid"`
	Problem        uuid.UUID   `json:"problem"`
	Model          uuid.UUID   `json:"model"` // @camillemarini: what's the diff. between model and algo ?
	Data           []uuid.UUID `json:"data"`
	Worker         uuid.UUID   `json:"worker"`
	Status         string      `json:"status"`
	RequestDate    time.Time   `json:"timestamp_request"`
	CompletionDate time.Time   `json:"timestamp_done"`
}

// Check returns nil if the preduplet is valid, an explicit error otherwise
func (u *Preduplet) Check() (err error) {

	if uuid.Equal(uuid.Nil, u.ID) {
		return fmt.Errorf("id field is unset")
	}

	if uuid.Equal(uuid.Nil, u.Problem) {
		return fmt.Errorf("problem field is unset")
	}

	if uuid.Equal(uuid.Nil, u.Model) {
		return fmt.Errorf("model field is required")
	}

	if len(u.Data) == 0 {
		return fmt.Errorf("data field is empty or unset")
	}
	for n, id := range u.Data {
		if uuid.Equal(uuid.Nil, id) {
			return fmt.Errorf("Nil UUID in data field at pos %d", n)
		}
	}

	if _, ok := ValidStatuses[u.Status]; !ok {
		return fmt.Errorf("status field ain't valid (provided: %s, possible choices: %s", u.Status, ValidStatuses)
	}

	return nil
}

// LearnUplet describes a Learning task.
type LearnUplet struct {
	Checkable

	ID             uuid.UUID   `json:"id"`
	Problem        uuid.UUID   `json:"problem"`
	TrainData      []uuid.UUID `json:"train_data"`
	TestData       []uuid.UUID `json:"test_data"`
	Algo           uuid.UUID   `json:"algo"`
	ModelStart     uuid.UUID   `json:"model_start"`
	ModelEnd       uuid.UUID   `json:"model_end"`
	Rank           int         `json:"rank"`
	WorkerID       uuid.UUID   `json:"worker"` // @camillemarini: I didn't get the purpose of this field
	Status         string      `json:"status"`
	Perf           float64     `json:"perf"`
	TrainPerf      float64     `json:"train_perf"`
	TestPerf       float64     `json:"test_perf"`
	RequestDate    time.Time   `json:"timestamp_request"`
	CompletionDate time.Time   `json:"timestamp_done"`
}

// Check returns nil if the learnuplet is valid, an explicit error otherwise
func (u *LearnUplet) Check() (err error) {

	if uuid.Equal(uuid.Nil, u.ID) {
		return fmt.Errorf("id field is required")
	}

	if uuid.Equal(uuid.Nil, u.Problem) {
		return fmt.Errorf("problem field is required")
	}

	if uuid.Equal(uuid.Nil, u.Algo) {
		return fmt.Errorf("algo field is required")
	}

	if len(u.TrainData) == 0 {
		return fmt.Errorf("train_data field is empty or unset")
	}
	for n, id := range u.TrainData {
		if uuid.Equal(uuid.Nil, id) {
			return fmt.Errorf("Nil UUID in train_data field at pos %d", n)
		}
	}

	if len(u.TestData) == 0 {
		return fmt.Errorf("test_data field is empty or unset")
	}
	for n, id := range u.TestData {
		if uuid.Equal(uuid.Nil, id) {
			return fmt.Errorf("Nil UUID in test_data field at pos %d", n)
		}
	}

	if _, ok := ValidStatuses[u.Status]; !ok {
		return fmt.Errorf("status field ain't valid (provided: %s, possible choices: %s", u.Status, ValidStatuses)
	}

	return nil
}

// APIError wraps errors sent back by the HTTP API
type APIError struct {
	Message string `json:"string"`
}

// NewAPIError creates an APIError object, given an error message
func NewAPIError(message string) (err *APIError) {
	return &APIError{
		Message: message,
	}
}

// TaskError describes an error happening in the consumer that indicates the errord task can be
// retried (if the retry limit hasn't been reached)
type TaskError struct {
	Message string `json:"string"`
}

func (e *TaskError) Error() string {
	return e.Message
}

// FatalTaskError describes an error happening in the consumer that isn't worth a retry
type FatalTaskError struct {
	Message string `json:"string"`
}

func (e *FatalTaskError) Error() string {
	return e.Message
}
