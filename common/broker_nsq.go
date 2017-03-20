package dccommon

import (
	"fmt"

	"github.com/nsqio/go-nsq"
)

const (
	BrokerNSQ = "nsq"
)

type ProducerNSQ struct {
	Producer

	NsqProducer *nsq.Producer
}

func NewNSQProducer(hostname string, port int) (p *ProducerNSQ, err error) {
	p = &ProducerNSQ{}

	config := nsq.NewConfig()
	p.NsqProducer, err = nsq.NewProducer(fmt.Sprintf("%s:%d%", hostname, port), config)
	if err != nil {
		return nil, fmt.Errorf("Error creating NSQ producer: %s", err)
	}

	return p, nil
}

func (p *ProducerNSQ) Push(topic string, body []byte) (err error) {
	err = p.NsqProducer.Publish(topic, body)
	if err != nil {
		return fmt.Errorf("Error publishing to NSQ: %s", err)
	}
	return nil
}

func (p *ProducerNSQ) Stop() {
	p.NsqProducer.Stop()
}
