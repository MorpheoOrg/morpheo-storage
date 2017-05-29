/*
 * Copyright Morpheo Org. 2017
 * 
 * contact@morpheo.co
 * 
 * This software is part of the Morpheo project, an open-source machine
 * learning platform.
 * 
 * This software is governed by the CeCILL license, compatible with the
 * GNU GPL, under French law and abiding by the rules of distribution of
 * free software. You can  use, modify and/ or redistribute the software
 * under the terms of the CeCILL license as circulated by CEA, CNRS and
 * INRIA at the following URL "http://www.cecill.info".
 * 
 * As a counterpart to the access to the source code and  rights to copy,
 * modify and redistribute granted by the license, users are provided only
 * with a limited warranty  and the software's author,  the holder of the
 * economic rights,  and the successive licensors  have only  limited
 * liability.
 * 
 * In this respect, the user's attention is drawn to the risks associated
 * with loading,  using,  modifying and/or developing or reproducing the
 * software by the user in light of its specific status of free software,
 * that may mean  that it is complicated to manipulate,  and  that  also
 * therefore means  that it is reserved for developers  and  experienced
 * professionals having in-depth computer knowledge. Users are therefore
 * encouraged to load and test the software's suitability as regards their
 * requirements in conditions enabling the security of their systems and/or
 * data to be ensured and,  more generally, to use and operate it in the
 * same conditions as regards security.
 * 
 * The fact that you are presently reading this means that you have had
 * knowledge of the CeCILL license and that you accept its terms.
 */

package common

import (
	"fmt"
	"time"

	"github.com/satori/go.uuid"
)

// Uplet types
const (
	TypeLearnUplet = "learnuplet"
	TypePredUplet  = "preduplet"
)

var (
	// ValidUplets us a set of all possible uplet names
	ValidUplets = map[string]struct{}{
		TypeLearnUplet: struct{}{},
		TypePredUplet:  struct{}{},
	}
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
	Model          uuid.UUID   `json:"model"`
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

	ID             uuid.UUID   `json:"uuid"`
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

// Error returns the error message as a string
func (err *APIError) Error() string {
	return err.Message
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

// Storage specific types

// Blob defines an abstract blob of data
type Blob struct {
	ID        uuid.UUID `json:"uuid" db:"uuid"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (b *Blob) fillNewBlob() {
	b.ID = uuid.NewV4()
	b.CreatedAt = time.Now()
}

// Problem defines a problem blob (should be a .tar.gz containing a Dockerfile)
type Problem struct {
	Blob

	Author uuid.UUID `json:"author" db:"author"`
}

// NewProblem creates a problem instance
func NewProblem() *Problem {
	problem := &Problem{
		Author: uuid.NewV4(),
	}
	problem.fillNewBlob()
	return problem
}

// Check returns nil for now
func (p *Problem) Check() error {
	// TODO: check what should be
	return nil
}

// Algo defines an algorithm blob (should be a .tar.gz containing a Dockerfile)
type Algo struct {
	Blob

	Author uuid.UUID `json:"author" db:"author"`
}

// NewAlgo creates an Algo instance
func NewAlgo() *Algo {
	algo := &Algo{
		Author: uuid.NewV4(),
	}
	algo.fillNewBlob()
	return algo
}

// Check returns nil for now
func (a *Algo) Check() error {
	// TODO: check what should be
	return nil
}

// Model defines a model blob (should be a .tar.gz of the model folder)
type Model struct {
	Blob

	Algo   uuid.UUID `json:"algo" db:"algo"`
	Author uuid.UUID `json:"author" db:"author"`
}

// NewModel creates a model instance
func NewModel(algo *Algo) *Model {
	model := &Model{
		Algo:   algo.ID,
		Author: uuid.NewV4(),
	}
	model.fillNewBlob()
	return model
}

// Check returns nil for now
func (m *Model) Check() error {
	// TODO: check what should be
	return nil
}

// Data defines a data blob
type Data struct {
	Blob

	Owner uuid.UUID `json:"owner" db:"owner"`
}

// NewData creates a problem instance
func NewData() *Data {
	data := &Data{
		Owner: uuid.NewV4(),
	}
	data.fillNewBlob()
	return data
}

// Check returns nil for now
func (d *Data) Check() error {
	// TODO: check what should be
	return nil
}
