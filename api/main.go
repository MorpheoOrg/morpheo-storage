package main

import (
	"encoding/json"
	"fmt"
	"log"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter" // <--- TODO or adaptors/gorillamux
	"gopkg.in/kataras/iris.v6/middleware/logger"

	"github.com/DeepSee/dc-compute"
)

func main() {
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

	// Handlers configuration
	app.Get("/", index)
	app.Post("/learn", postLearnuplet)
	app.Post("/pred", postPreduplet)
	app.Post("/learn_task", postLearnTask)
	app.Post("/test_task", postTestTask)

	// Main server loop
	app.Listen("0.0.0.0:8282")
}

func index(c *iris.Context) {
	c.JSON(iris.StatusOK, []string{"/learn", "/pred", "/learn_task", "/test_task"})
}

func postLearnuplet(c *iris.Context) {
	var learnUplet dccompute.LearnUplet

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&learnUplet); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}

	// Let's check for required arguments presence and validity
	if err := learnUplet.Check(); err != nil {
		msg := fmt.Sprintf("Invalid learn-uplet: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}

	firstTask := learnUplet.SplitTrain()

	log.Println(firstTask.Next.Data.String())
	// TODO: notify the orchestrator we're starting this learning process
	// TODO: ask storage for the cluster to send the first task to (shouldn't it rather be set by the
	// orchestrator ?)
	// TODO: send that damn task to the appropriate compute cluster
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Learn-uplet Ingested"})
}

func postPreduplet(c *iris.Context) {
	var predUplet dccompute.Preduplet

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&predUplet); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}

	// Let's check for required arguments presence and validity
	if err := predUplet.Check(); err != nil {
		msg := fmt.Sprintf("Invalid pred-uplet: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}
	// TODO: notify the orchestrator we're starting this prediction process
	// TODO: send task to broker
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Pred-uplet ingested"})
}

func postLearnTask(c *iris.Context) {
	var learnTask dccompute.LearnTask

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&learnTask); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}

	// Let's check for required arguments presence and validity
	if err := learnTask.Check(); err != nil {
		msg := fmt.Sprintf("Invalid learn-task: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}
	// TODO: notify the orchestrator we're starting this prediction process
	// TODO: send task to broker
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Learn task ingested"})
}

func postTestTask(c *iris.Context) {
	var testTask dccompute.TestTask

	// Unserializing the request body
	if err := json.NewDecoder(c.Request.Body).Decode(&testTask); err != nil {
		msg := fmt.Sprintf("Error decoding body to JSON: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}

	// Let's check for required arguments presence and validity
	if err := testTask.Check(); err != nil {
		msg := fmt.Sprintf("Invalid test-task: %s", err)
		log.Printf("[INFO] %s", msg)
		c.JSON(iris.StatusBadRequest, map[string]string{"message": msg})
		return
	}
	// TODO: notify the orchestrator we're starting this prediction process
	// TODO: send task to broker
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Test task ingested"})
}
