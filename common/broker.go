package dccommon

// Interface to a producer (pushes messages to a topic)
type Producer interface {
	Push(topic string, body []byte) (err error)
	Stop()
}
