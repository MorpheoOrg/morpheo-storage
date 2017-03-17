package dccompute

import (
	"fmt"

	"github.com/satori/go.uuid"
)

// Interface for things that can be Checked (i.e. validated after a JSON
// parsing for instance)
type Checkable interface {
	Check() (err error)
}

// Defines a pred-uplet
type Preduplet struct {
	Checkable

	Id      uuid.UUID   `json:"id"`
	Problem uuid.UUID   `json:"problem"`
	Model   uuid.UUID   `json:"model"`
	Data    []uuid.UUID `json:"data"` // ? Why a list field here ?
}

func (u *Preduplet) Check() (err error) {

	if uuid.Equal(uuid.Nil, u.Id) {
		return fmt.Errorf("id field is unset")
	}

	if uuid.Equal(uuid.Nil, u.Problem) {
		return fmt.Errorf("problem field is unset")
	}

	if uuid.Equal(uuid.Nil, u.Model) {
		return fmt.Errorf("model field is required")
	}

	if len(u.Data) == 0 {
		return fmt.Errorf("train_data is empty or unset")
	}
	for _, id := range u.Data {
		if uuid.Equal(uuid.Nil, id) {
			return fmt.Errorf("Nil UUID in dataset")
		}
	}

	return nil
}

// Defines a learn-uplet
type LearnUplet struct {
	Checkable
	// TODO add rank

	Id             uuid.UUID   `json:"id"`
	Problem        uuid.UUID   `json:"problem"`
	Model          uuid.UUID   `json:"model"`
	LearnDataset   []uuid.UUID `json:"train_data"`
	TestDataset    []uuid.UUID `json:"test_data"`
	IndividualPerf []float64   `json:"individual_perf"`
}

func (u *LearnUplet) Check() (err error) {

	if uuid.Equal(uuid.Nil, u.Id) {
		return fmt.Errorf("id field is required")
	}

	if uuid.Equal(uuid.Nil, u.Problem) {
		return fmt.Errorf("problem field is required")
	}

	if uuid.Equal(uuid.Nil, u.Model) {
		return fmt.Errorf("model field is required")
	}

	if len(u.LearnDataset) == 0 {
		return fmt.Errorf("train_data is empty or unset")
	}
	for _, id := range u.LearnDataset {
		if uuid.Equal(uuid.Nil, id) {
			return fmt.Errorf("Nil UUID in LearnDataset")
		}
	}
	if len(u.TestDataset) == 0 {
		return fmt.Errorf("test_data is empty or unset")
	}
	for _, id := range u.TestDataset {
		if uuid.Equal(uuid.Nil, id) {
			return fmt.Errorf("Nil UUID in LearnDataset")
		}
	}

	return nil
}

// Splits the train-uplet into a linked list of LearnTasks
func (u *LearnUplet) SplitTrain() (firstTask *LearnTask) {
	u.nest(&firstTask, u.LearnDataset)
	return
}

// A tiny little bit of tail recursion doesn't hurt once in a while
func (u *LearnUplet) nest(ptr **LearnTask, slice []uuid.UUID) {
	if len(slice) <= 0 {
		return
	}

	*ptr = &LearnTask{
		Data:       slice[0],
		LearnUplet: u,
	}
	u.nest(&(*ptr).Next, slice[1:])
}

// Defines a tearning task on one unit of data
type LearnTask struct {
	Checkable

	Data       uuid.UUID   `json:"data"`
	Next       *LearnTask  `json:"next"`
	LearnUplet *LearnUplet `json:learn-uplet`
}

func (u *LearnTask) Check() (err error) {

	if uuid.Equal(uuid.Nil, u.Data) {
		return fmt.Errorf("data field is unset")
	}

	return nil
}

// Defines a test task
type TestTask struct {
	Checkable

	Data       uuid.UUID   `json:"data"`
	Next       *TestTask   `json:"next"`
	LearnUplet *LearnUplet `json:"learn-uplet"`
}

func (u *TestTask) Check() (err error) {

	if uuid.Equal(uuid.Nil, u.Data) {
		return fmt.Errorf("data field is unset")
	}

	return nil
}

// This object is sent back by the HTTP API on errors
type APIError struct {
	Message string `json:"string"`
}

func NewAPIError(message string) (err *APIError) {
	return &APIError{
		Message: message,
	}
}
