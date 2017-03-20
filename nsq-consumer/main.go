package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/nsqio/go-nsq"

	"github.com/DeepSee/dc-compute"
	common "github.com/DeepSee/dc-compute/common"
)

type MessageHandler struct {
	nsq.Handler

	Topic   string
	Channel string
	backend common.ContainerBackend
}

func (h *MessageHandler) HandleMessage(message *nsq.Message) error {
	// TODO: disable requeuing on specific errors
	// TODO: maybe we could implement an interface that wraps around different types of tasks

	// Let's run our ML task
	var err error
	switch h.Topic {
	case dccompute.LearnTopic:
		var task dccompute.LearnTask
		err = json.NewDecoder(bytes.NewReader(message.Body)).Decode(&task)
		if err != nil {
			message.Finish()
			return fmt.Errorf("Error un-marshaling train task: %s -- Body: %s", err, message.Body)
		}

		if err = task.Check(); err != nil {
			message.Finish()
			return fmt.Errorf("Error un-marshaling train task: %s -- Body: %s", err, message.Body)
		}

		err = h.backend.RunInUntrustedContainer(task.LearnUplet.Model.String(), h.Topic, task.Data.String())

	case dccompute.TestTopic:
		var task dccompute.TestTask
		err = json.NewDecoder(bytes.NewReader(message.Body)).Decode(&task)
		if err != nil {
			return fmt.Errorf("Error un-marshaling test task: %s -- Body: %s", err, message.Body)
		}

		err = h.backend.RunInUntrustedContainer(task.LearnUplet.Model.String(), h.Topic, task.Data.String())

	case dccompute.PredictionTopic:
		var task dccompute.Preduplet
		err = json.NewDecoder(bytes.NewReader(message.Body)).Decode(&task)
		if err != nil {
			return fmt.Errorf("Error un-marshaling pred-uplet: %s -- Body: %s", err, message.Body)
		}

		err = h.backend.RunInUntrustedContainer(task.Model.String(), h.Topic, task.Data.String())
	default:
		return fmt.Errorf("Topic %s unkown to consumer.", h.Topic)
	}

	if err != nil {
		return fmt.Errorf("Container backend error: %s", err)
	}

	log.Printf("Successfully completed task !")
	return nil
}

func main() {
	// TODO: improve config and add a -container-backend flag
	var (
		lookupUrls dccompute.MultiStringFlag
		topic      string
		channel    string
	)

	flag.Var(&lookupUrls, "lookup-urls", "The URLs of the Nsqlookupd instances to fetch our topics from.")
	flag.StringVar(&topic, "topic", "learn", "The hostname of the Nsqd/Nsqlookupd instance to get messages from")
	flag.StringVar(&channel, "channel", "compute", "The channel to use (default: compute)")
	flag.Parse()

	// Config check
	if len(lookupUrls) == 0 {
		lookupUrls = append(lookupUrls, "nsqlookupd:6460")
	}

	if topic != dccompute.LearnTopic && topic != dccompute.PredictionTopic && topic != dccompute.TestTopic {
		log.Panicf("Unknown topic: %s, valid values are %s, %s and %s", topic, dccompute.LearnTopic, dccompute.TestTopic, dccompute.PredictionTopic)
	}

	// Consumer creation
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		log.Panicf("Impossible to create NSQ consumer: %s", err)
	}

	// Let's hook to our container backend
	containerBackend, err := common.NewDockerBackend("file://var/run/docker.sock")
	if err != nil {
		log.Panicf("Impossible to connect to Docker container backend: %s", err)
	}

	// Wire our message Handler
	h := &MessageHandler{
		Topic:   topic,
		Channel: channel,
		backend: containerBackend,
	}
	consumer.AddHandler(h)

	// Let's connect to NSQd for real and start pulling tasks
	for {
		err = consumer.ConnectToNSQLookupds(lookupUrls)
		if err == nil {
			break
		}

		log.Printf("[nsqlookupd-warning]: %s", err)
		time.Sleep(1 * time.Second) // TODO: put that in the config
	}
	log.Println("[nsqlookupd] Topic found, let's start consuming messages...")

	// Using a channel for this purpose is a bit weird... but why not Bitly guys :)
	<-consumer.StopChan

	log.Println("Consumer has been gracefully stopped... Bye bye!")
	return
}
