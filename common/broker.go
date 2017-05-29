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
)

// Topics (task queue names) for our broker
const (
	TrainTopic   = "train"
	PredictTopic = "prediction"
)

// Producer is an abstract interface to a producer (pushes messages to a topic)
type Producer interface {
	Push(topic string, body []byte) (err error)
	Stop()
}

// Consumer is an abstract interface to a consumer (consumes messages from a topic). One
// implementation per broker is possible.
type Consumer interface {
	// Consume messages from the selected topic continuously.  It also sound like a relevant motto for
	// the society we live in :)
	ConsumeUntilKilled()

	// Add a handler function to the consumer for a given topic name. Up to concurrency tasks will be
	// executed in parrallel. After the given timeout is reached, the task will be considered failed
	// and will be re-enqueued.
	AddHandler(topic string, handler Handler, concurrency int, timeout time.Duration)
}

// Handler is an abstract Interface to a message handler Abstracts the way messages are handled so
// that different handlers can easily be passed for different topics
type Handler func(message []byte) error

// HandlerFatalError is a simple wrapper type around fatal handler errors. If a fatal error occurred
// during the handling of a message, the latter won't be requeued.
// TODO: try and unit test the behaviour of this interface
type HandlerFatalError struct {
	message string
}

func (err HandlerFatalError) Error() string {
	return fmt.Sprintf("Fatal error in handler: ")
}

// NewHandlerFatalError builds an HandlerFatalError given an error message
func NewHandlerFatalError(err error) HandlerFatalError {
	return HandlerFatalError{
		message: err.Error(),
	}
}
