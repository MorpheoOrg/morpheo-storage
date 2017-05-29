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
	"log"
	"time"

	"github.com/nsqio/go-nsq"
)

const (
	// BrokerNSQ identifies the NSQ broker type among other brokers (used when the user specifies the
	// broker to be used as a CLI flag)
	BrokerNSQ = "nsq"
)

// ProducerNSQ is an implementation of our Producer interface for NSQ
type ProducerNSQ struct {
	Producer

	NsqProducer *nsq.Producer
}

// NewNSQProducer creates an instance of NSQProducer. Produced messages are sent to an Nsqd instance
// accessible under the given (host, port) TCP/IP destination
func NewNSQProducer(hostname string, port int) (p *ProducerNSQ, err error) {
	p = &ProducerNSQ{}

	config := nsq.NewConfig()
	p.NsqProducer, err = nsq.NewProducer(fmt.Sprintf("%s:%d", hostname, port), config)
	if err != nil {
		return nil, fmt.Errorf("Error creating NSQ producer: %s", err)
	}

	return p, nil
}

// Push sends a message to the nsqd instance bound to p under a given topic
func (p *ProducerNSQ) Push(topic string, body []byte) (err error) {
	err = p.NsqProducer.Publish(topic, body)
	if err != nil {
		return fmt.Errorf("Error publishing to NSQ: %s", err)
	}
	return nil
}

// Stop stops the NSQProducer instances (no more messages will be forwarded to nsqd)
func (p *ProducerNSQ) Stop() {
	p.NsqProducer.Stop()
}

// ConsumerNSQ implements an NSQ version of our Consumer interface
type ConsumerNSQ struct {
	Consumer

	NsqConsumer          map[string]*nsq.Consumer
	LookupUrls           []string
	QueuePollingInterval time.Duration
	Channel              string
}

// NewNSQConsumer instantiates ConsumerNSQ for the provided channel, using provided nsqlookupd URLs
func NewNSQConsumer(lookupUrls []string, channel string, queuePollingInterval time.Duration) (c *ConsumerNSQ) {
	return &ConsumerNSQ{
		LookupUrls:           lookupUrls,
		Channel:              channel,
		QueuePollingInterval: queuePollingInterval,
		NsqConsumer:          map[string]*nsq.Consumer{},
	}
}

// ConsumeUntilKilled listens for messages on a given NSQ (topic, channel) pair until it's killed
func (c *ConsumerNSQ) ConsumeUntilKilled() {
	for _, consumer := range c.NsqConsumer {
		go func(nsqConsumer *nsq.Consumer) {
			for {
				err := nsqConsumer.ConnectToNSQLookupds(c.LookupUrls)
				if err == nil {
					break
				}

				log.Printf("[nsqlookupd-warning]: %s", err)
				time.Sleep(c.QueuePollingInterval)
			}
			log.Println("[nsqlookupd] Topic found, let's start consuming messages...")
		}(consumer)
	}

	// Let's block until all the consumers stop
	for _, consumer := range c.NsqConsumer {
		// Using a channel for this purpose is a bit weird... but why not Bitly guys :)
		<-consumer.StopChan
	}
}

// AddHandler adds a handler function (with a tunable level of concurrency) to our NSQ consumer
func (c *ConsumerNSQ) AddHandler(topic string, handler Handler, concurrency int, timeout time.Duration) (err error) {
	log.Printf("Adding %d handler(s) for topic %s.", concurrency, topic)

	// Let's add our handler to that (topic, channel) tuple
	config := nsq.NewConfig()
	config.LookupdPollInterval = c.QueuePollingInterval
	config.MaxAttempts = 1
	config.HeartbeatInterval = c.QueuePollingInterval
	config.MsgTimeout = timeout

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
	log.Printf("[DEBUG] nsq-consumer received task")
	err = hw.handler(message.Body)
	// TODO: smart backoff strategy
	// if _, fatal := err.(HandlerFatalError); fatal {
	message.Finish()
	// }
	return err
}
