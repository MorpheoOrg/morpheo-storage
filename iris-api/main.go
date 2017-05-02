package main

import (
	"encoding/json"
	"fmt"
	"log"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter" // <--- TODO or adaptors/gorillamux
	"gopkg.in/kataras/iris.v6/middleware/logger"

	"github.com/DeepSee/dc-compute"
	common "github.com/DeepSee/dc-compute/common"
)

// TODO: write tests for the four main views

// Available HTTP Routes
const (
	rootRoute   = "/"
	healthRoute = "/health"
	learnRoute  = "/learn"
	predRoute   = "/pred"
)

type apiServer struct {
	conf     *dccompute.ProducerConfig
	producer common.Producer
}

func newAPIServer(conf *dccompute.ProducerConfig, producer common.Producer) (s *apiServer) {
	return &apiServer{
		conf:     conf,
		producer: producer,
	}
}

func (s *apiServer) configureRoutes(app *iris.Framework) {
	app.Get(rootRoute, s.index)
	app.Get(healthRoute, s.health)
	app.Post(learnRoute, s.postLearnuplet)
	app.Post(predRoute, s.postPreduplet)
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
	apiServer := newAPIServer(conf, producer)
	apiServer.configureRoutes(app)

	// Main server loop
	if conf.TLSOn() {
		app.ListenTLS(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port), conf.CertFile, conf.KeyFile)
	} else {
		app.Listen(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port))
	}
}

func (s *apiServer) index(c *iris.Context) {
	c.JSON(iris.StatusOK, []string{"/learn", "/pred", "/learn_task", "/test_task"})
}

func (s *apiServer) health(c *iris.Context) {
	c.JSON(iris.StatusOK, map[string]string{"status": "ok"})
}

func (s *apiServer) postLearnuplet(c *iris.Context) {
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

	// Let's put our LearnUplet in the right topic so that it gets processed for real
	taskBytes, err := json.Marshal(learnUplet)
	if err != nil {
		msg := fmt.Sprintf("Failed to remarshal JSON learn-uplet after validation: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}
	err = s.producer.Push(dccompute.LearnTopic, taskBytes)
	if err != nil {
		msg := fmt.Sprintf("Failed push learn-uplet into broker: %s", err)
		log.Printf("[ERROR] %s", msg)
		c.JSON(iris.StatusInternalServerError, dccompute.NewAPIError(msg))
		return
	}

	// TODO: notify the orchestrator we're starting this learning process (using the Go orchestrator
	// API). We can either do a PATCH the status field or re-PUT the whole learnuplet (since it has
	// already been computed and is stored in variable taskBytes)

	c.JSON(iris.StatusAccepted, map[string]string{"message": "Learn-uplet ingested accordingly"})
}

func (s *apiServer) postPreduplet(c *iris.Context) {
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

	// TODO: notify the orchestrator we're starting this learning process (using the Go orchestrator
	// API). We can either do a PATCH the status field or re-PUT the whole learnuplet (since it has
	// already been computed and is stored in variable taskBytes)

	c.JSON(iris.StatusAccepted, map[string]string{"message": "Pred-uplet ingested"})
}
