package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter" // <--- TODO or adaptors/gorillamux
	"gopkg.in/kataras/iris.v6/middleware/logger"

	"github.com/DeepSee/dc-compute"
	common "github.com/DeepSee/dc-compute/common"
)

// TODO: write tests for the four main views

const (
	rootRoute      = "/"
	healthRoute    = "/health"
	learnRoute     = "/learn"
	predRoute      = "/pred"
	learnTaskRoute = "/learn-task"
	testTaskRoute  = "/test-task"
)

type APIServer struct {
	conf     *dccompute.ProducerConfig
	producer common.Producer
}

func NewAPIServer(conf *dccompute.ProducerConfig, producer common.Producer) (s *APIServer) {
	return &APIServer{
		conf:     conf,
		producer: producer,
	}
}

func (s *APIServer) configureRoutes(app *iris.Framework) {
	app.Get(rootRoute, s.index)
	app.Get(healthRoute, s.health)
	app.Post(learnRoute, s.postLearnuplet)
	app.Post(predRoute, s.postPreduplet)
	app.Post(learnTaskRoute, s.postLearnTask)
	app.Post(testTaskRoute, s.postTestTask)
}

func main() {
	// App-specific config
	conf := dccompute.NewProducerConfig()

	// Iris setup
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(httprouter.New())

	// Logger middleware configuration
	customLogger := logger.New(logger.Config{
		Status: true,
		IP:     true,
		Method: true,
		Path:   true,
	})
	app.Use(customLogger)

	// Let's dependency inject the producer for the chosen Broker
	var producer common.Producer
	switch conf.Broker {
	case common.BrokerNSQ:
		var err error
		producer, err = common.NewNSQProducer(conf.BrokerHost, conf.BrokerPort)
		defer producer.Stop()
		if err != nil {
			log.Panicln(err)
		}
	default:
		log.Panicf("Unsupported broker (%s). The only available broker is 'nsq'", conf.Broker)
	}

	// Handlers configuration
	apiServer := NewAPIServer(conf, producer)
	apiServer.configureRoutes(app)

	// Main server loop
	if conf.TLSOn() {
		app.ListenTLS(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port), conf.CertFile, conf.KeyFile)
	} else {
		app.Listen(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port))
	}
}

func (s *APIServer) index(c *iris.Context) {
	c.JSON(iris.StatusOK, []string{"/learn", "/pred", "/learn_task", "/test_task"})
}

func (s *APIServer) health(c *iris.Context) {
	c.JSON(iris.StatusOK, map[string]string{"status": "ok"})
}

func (s *APIServer) postLearnuplet(c *iris.Context) {
	var learnUplet dccompute.LearnUplet

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&learnUplet); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}

	// Let's check for required arguments presence and validity
	if err := learnUplet.Check(); err != nil {
		msg := fmt.Sprintf("Invalid learn-uplet: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}

	firstTask := learnUplet.SplitTrain()

	// TODO: notify the orchestrator we're starting this learning process (using the Go orchestrator
	// API)
	// TODO: ask storage for the cluster to send the first task to (shouldn't it rather be set by the
	// orchestrator ?) --> need of a Golang starage API maybe
	executionClusterURL := "http://localhost:8085/learn-task"
	// TODO: send that damn task to the appropriate compute cluster

	payload, err := json.Marshal(firstTask)
	if err != nil {
		msg := fmt.Sprintf("Failed to Marshal first task: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}

	req, err := http.NewRequest("POST", executionClusterURL, bytes.NewBuffer(payload))
	if err != nil {
		msg := fmt.Sprintf("Failed to build first task POST request for compute cluster [%s]: %s", executionClusterURL, err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Destination compute cluster [%s] refused first task: %s", executionClusterURL, err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}

	if resp.StatusCode != iris.StatusAccepted {
		var apiError dccompute.APIError
		err := json.NewDecoder(c.Request.Body).Decode(&apiError)
		if err != nil && err != io.EOF {
			msg := fmt.Sprintf("Failed to parse [%s] JSON response: %s", executionClusterURL, err)
			log.Printf("[ERROR] %s", msg)
			c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
			return
		}
		msg := fmt.Sprintf("Task refused by [%s] - Status Code: %d - Message: %s", executionClusterURL, resp.StatusCode, apiError.Message)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}

	c.JSON(iris.StatusAccepted, map[string]string{"message": "Learn-uplet ingested accordingly"})
}

func (s *APIServer) postPreduplet(c *iris.Context) {
	var predUplet dccompute.Preduplet

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&predUplet); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}

	// Let's check for required arguments presence and validity
	if err := predUplet.Check(); err != nil {
		msg := fmt.Sprintf("Invalid pred-uplet: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}
	// TODO: notify the orchestrator we're starting this prediction process
	taskBytes, err := json.Marshal(predUplet)
	if err != nil {
		msg := fmt.Sprintf("Failed to remarshal preduplet to JSON: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}
	err = s.producer.Push(dccompute.PredictionTopic, taskBytes)
	if err != nil {
		msg := fmt.Sprintf("Failed to push preduplet task into broker: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Pred-uplet ingested"})
}

func (s *APIServer) postLearnTask(c *iris.Context) {
	var learnTask dccompute.LearnTask

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&learnTask); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}

	// Let's check for required arguments presence and validity
	if err := learnTask.Check(); err != nil {
		msg := fmt.Sprintf("Invalid learn-task: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}
	// TODO: notify the orchestrator we're starting this prediction process
	taskBytes, err := json.Marshal(learnTask)
	if err != nil {
		msg := fmt.Sprintf("Failed to remarshal JSON learn task: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}
	err = s.producer.Push(dccompute.LearnTopic, taskBytes)
	if err != nil {
		msg := fmt.Sprintf("Failed push learn task into broker: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Learn task ingested"})
}

func (s *APIServer) postTestTask(c *iris.Context) {
	var testTask dccompute.TestTask

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&testTask); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}

	// Let's check for required arguments presence and validity
	if err := testTask.Check(); err != nil {
		msg := fmt.Sprintf("Invalid test task: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, dccompute.NewAPIError(msg))
		return
	}
	// TODO: notify the orchestrator we're starting this prediction process
	taskBytes, err := json.Marshal(testTask)
	if err != nil {
		msg := fmt.Sprintf("Failed to remarshal test task to JSON: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}
	err = s.producer.Push(dccompute.TestTopic, taskBytes)
	if err != nil {
		msg := fmt.Sprintf("Failed to push test task into broker: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Test task ingested"})
}
