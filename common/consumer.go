package dccommon

import "fmt"

// Interface to a consumer (consumes messages from a topic)
type Consumer interface {
	// Consume messages from the selected topic continuously.
	// It also sound like a relevant motto for the society we live in :)
	ConsumeUntilKilled()

	// Add a handler function to the consumer for a given topic name. Up to
	// concurrency tasks will be executed in parrallel
	AddHandler(topic string, handler Handler, concurrency int)
}

// Interface to a message handler
// Abstracts the way messages are handled so that different handlers can
// easily be passed for different topics
type Handler func(message []byte) error

// A simple wrapper type around fatal handler errors. If a fatal error occurred
// during the handling of a message, the latter won't be requeued.
// TODO: try and unit test the behaviour of this interface
type HandlerFatalError struct {
	message string
}

func (err HandlerFatalError) Error() string {
	return fmt.Sprintf("Fatal error in handler: %s", err)
}

func NewHandlerFatalError(err error) HandlerFatalError {
	return HandlerFatalError{
		message: err.Error(),
	}
}
