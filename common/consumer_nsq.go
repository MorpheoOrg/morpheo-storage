package dccommon

import (
	"fmt"
	"log"
	"time"

	"github.com/nsqio/go-nsq"
)

type ConsumerNSQ struct {
	Consumer

	NsqConsumer          map[string]*nsq.Consumer
	LookupUrls           []string
	QueuePollingInterval time.Duration
	Channel              string
}

func NewNSQConsumer(lookupUrls []string, channel string, queuePollingInterval time.Duration) (c *ConsumerNSQ) {
	return &ConsumerNSQ{
		LookupUrls:           lookupUrls,
		Channel:              channel,
		QueuePollingInterval: queuePollingInterval,
		NsqConsumer:          map[string]*nsq.Consumer{},
	}
}

func (c *ConsumerNSQ) ConsumeUntilKilled() {
	for _, consumer := range c.NsqConsumer {
		go func() {
			for {
				err := consumer.ConnectToNSQLookupds(c.LookupUrls)
				if err == nil {
					break
				}

				log.Printf("[nsqlookupd-warning]: %s", err)
				time.Sleep(c.QueuePollingInterval)
			}
			log.Println("[nsqlookupd] Topic found, let's start consuming messages...")
		}()
	}

	// Let's block until all the consumers stop
	for _, consumer := range c.NsqConsumer {
		// Using a channel for this purpose is a bit weird... but why not Bitly guys :)
		<-consumer.StopChan
	}
}

func (c *ConsumerNSQ) AddHandler(topic string, handler Handler, concurrency int) (err error) {
	log.Printf("Adding %d handler(s) for topic %s.", concurrency, topic)
	// Let's add our handler to that (topic, channel) tuple
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(topic, c.Channel, config)
	if err != nil {
		return fmt.Errorf("Error creating NSQ Consumer for topic %s: %s", topic, err)
	}
	consumer.AddConcurrentHandlers(newHandlerWrapper(handler), concurrency)
	c.NsqConsumer[topic] = consumer

	return nil
}

type handlerWrapper struct {
	nsq.Handler

	handler Handler
}

func newHandlerWrapper(handler Handler) *handlerWrapper {
	return &handlerWrapper{
		handler: handler,
	}
}

func (hw *handlerWrapper) HandleMessage(message *nsq.Message) (err error) {
	err = hw.handler(message.Body)
	if _, fatal := err.(HandlerFatalError); fatal {
		message.Finish()
	}
	return
}
