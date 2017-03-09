package dccompute

import "github.com/satori/go.uuid"

// Defines a pred-uplet
type Preduplet struct {
	Id      uuid.UUID   `json:"id"`
	Problem uuid.UUID   `json:"problem"`
	Data    []uuid.UUID `json:"data"` // ? Why a list field here ?
	Model   uuid.UUID   `json:"model"`
}

// Defines a learn-uplet
type LearnUplet struct {
	Id             uuid.UUID   `json:"id"`
	Problem        uuid.UUID   `json:"problem"`
	LearnDataset   []uuid.UUID `json:"train_data"`
	TestDataset    []uuid.UUID `json:"test_data"`
	IndividualPerf []float64   `json:"individual_perf"`
}

// Splits the train-uplet into a linked list of LearnTasks
func (u *LearnUplet) SplitTrain() (firstTask *LearnTask) {
	nest(firstTask, u.TrainDataset)
	return
}

// A tiny little bit of tail recursion doesn't hurt once in a while
func nest(ptr *LearnTask, slice []uuid.UUID) {
	if len(slice) <= 0 {
		return
	}

	ptr = &LearnTask{
		Data:       slice[0],
		LearnUplet: &u,
	}
	nest(ptr.Next, slice[1:])
}

// Defines a tearning task on one unit of data
type LearnTask struct {
	Data       uuid.UUID   `json:"data"`
	Next       *LearnTask  `json:"next"`
	LearnUplet *LearnUplet `json:learn-uplet`
}

// Defines a test task
type TestTask struct {
	Data       uuid.UUID   `json:"data"`
	Next       *TestTask   `json:"next"`
	LearnUplet *LearnUplet `json:"learn-uplet"`
}
