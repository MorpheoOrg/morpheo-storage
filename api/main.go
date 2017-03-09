package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter" // <--- or adaptors/gorillamux
)

func main() {
	app := iris.New()
	app.Adapt(httprouter.New())

	// Handlers configuration
	app.Get("/", index)
	app.Post("/train", postTrainuplet)
	app.Post("/pred", postPreduplet)
	app.Post("/train_task", postTrainTask)
	app.Post("/test_task", postTestTask)

	// Main server loop
	app.Listen("0.0.0.0:8282")
}

func index(c *iris.Context) {
	c.JSON(iris.StatusOK, []string{"train", "predict"})
}

func postTrainuplet(c *iris.Context) {
	// TODO: parse that
	// TODO: split learnuplet into LearnTasks and TestTask
	// TODO: notify the orchestrator we're starting this learning process
	// TODO: ask storage for the cluster to send the first task to (shouldn't it rather be set by the
	// orchestrator ?)
	// TODO: send that damn task
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Train-uplet Ingested"})
}

func postPreduplet(c *iris.Context) {
	// TODO: parse that
	// TODO: notify the orchestrator we're starting this prediction process
	// TODO: send that damn task
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Pred-uplet ingested"})
}

func postTrainTask(c *iris.Context) {
	// TODO: parse that
	// TODO: notify the orchestrator we're starting this prediction process
	// TODO: send that damn task
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Train task ingested"})
}

func postTestTask(c *iris.Context) {
	// TODO: parse that
	// TODO: notify the orchestrator we're starting this prediction process
	// TODO: send that damn task
	c.JSON(iris.StatusAccepted, map[string]string{"message": "Test task ingested"})
}
