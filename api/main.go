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

package main

import (
	"fmt"

	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/satori/go.uuid"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/cors"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter" // <--- TODO or adaptors/gorillamux
	"gopkg.in/kataras/iris.v6/middleware/basicauth"
	"gopkg.in/kataras/iris.v6/middleware/logger"

	"github.com/MorpheoOrg/morpheo-go-packages/common"
)

// Available HTTP routes
const (
	RootRoute           = "/"
	HealthRoute         = "/health"
	ProblemListRoute    = "/problem"
	ProblemRoute        = "/problem/:uuid"
	ProblemBlobRoute    = "/problem/:uuid/blob"
	DataListRoute       = "/data"
	DataRoute           = "/data/:uuid"
	DataBlobRoute       = "/data/:uuid/blob"
	AlgoListRoute       = "/algo"
	AlgoRoute           = "/algo/:uuid"
	AlgoBlobRoute       = "/algo/:uuid/blob"
	ModelListRoute      = "/model"
	ModelRoute          = "/model/:uuid"
	ModelBlobRoute      = "/model/:uuid/blob"
	PredictionListRoute = "/prediction"
	PredictionRoute     = "/prediction/:uuid"
	PredictionBlobRoute = "/prediction/:uuid/blob"
)

// APIServer represents the API configurations
type APIServer struct {
	Conf            *StorageConfig
	BlobStore       common.BlobStore
	ProblemModel    Model
	AlgoModel       Model
	ModelModel      Model
	DataModel       Model
	PredictionModel Model
}

// ConfigureRoutes links the urls with the func and set authentication
func (s *APIServer) ConfigureRoutes(app *iris.Framework, authentication iris.HandlerFunc) {
	// Misc.
	app.Get(RootRoute, s.index)
	app.Get(HealthRoute, s.health)

	// Problem
	app.Get(ProblemListRoute, authentication, s.getProblemList)
	app.Post(ProblemListRoute, authentication, s.postProblem)
	app.Patch(ProblemRoute, authentication, s.patchProblem)
	app.Get(ProblemRoute, authentication, s.getProblem)
	app.Get(ProblemBlobRoute, authentication, s.getProblemBlob)

	// Algo
	app.Get(AlgoListRoute, authentication, s.getAlgoList)
	app.Post(AlgoListRoute, authentication, s.postAlgo)
	app.Get(AlgoRoute, authentication, s.getAlgo)
	app.Get(AlgoBlobRoute, authentication, s.getAlgoBlob)

	// Model
	app.Get(ModelListRoute, authentication, s.getModelList)
	app.Post(ModelListRoute, authentication, s.postModel)
	app.Get(ModelRoute, authentication, s.getModel)
	app.Get(ModelBlobRoute, authentication, s.getModelBlob)

	// Data
	app.Get(DataListRoute, authentication, s.getDataList)
	app.Post(DataListRoute, authentication, s.postData)
	app.Get(DataRoute, authentication, s.getData)
	app.Get(DataBlobRoute, authentication, s.getDataBlob)

	// Prediction
	app.Get(PredictionListRoute, authentication, s.getPredictionList)
	app.Post(PredictionListRoute, authentication, s.postPrediction)
	app.Get(PredictionRoute, authentication, s.getPrediction)
	app.Get(PredictionBlobRoute, authentication, s.getPredictionBlob)
}

// RunMigrations applies migrations in migrationDir
func RunMigrations(db *sqlx.DB, migrationDir string, rollback bool) (int, error) {
	migrate.SetTable(migrationTable)

	migrations := &migrate.FileMigrationSource{
		Dir: migrationDir,
	}

	operation := migrate.Up
	limit := 0
	if rollback {
		limit = 1
		operation = migrate.Down
	}

	return migrate.ExecMax(db.DB, "postgres", migrations, operation, limit)
}

// SetAuthentication returns the app authentication
func SetAuthentication(user, password string) iris.HandlerFunc {
	authConfig := basicauth.Config{
		Users:      map[string]string{user: password},
		Realm:      "Authorization Required",
		ContextKey: "mycustomkey",
		Expires:    time.Duration(30) * time.Minute,
	}
	return basicauth.New(authConfig)
}

func main() {
	// Parses CLI flags to generate the API config
	conf := NewStorageConfig()

	// Iris setup
	app := iris.New()
	app.Adapt(iris.DevLogger(), httprouter.New())

	// Iris authentication
	authentication := SetAuthentication(conf.APIUser, conf.APIPassword)

	// Iris CORS middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	app.Adapt(corsMiddleware)

	// Logging middleware configuration
	customLogger := logger.New(logger.Config{
		Status: true,
		IP:     true,
		Method: true,
		Path:   true,
	})
	app.Use(customLogger)
	db, err := sqlx.Connect(
		"postgres",
		fmt.Sprintf(
			"user=%s password=%s host=%s port=%d sslmode=disable dbname=%s",
			conf.DBUser, conf.DBPass, conf.DBHost, conf.DBPort, conf.DBName,
		),
	)
	if err != nil {
		log.Fatalf("Cannot open connection to database: %s", err)
	}

	n, err := RunMigrations(db, conf.DBMigrationsDir, conf.DBRollback)
	if err != nil {
		log.Fatalf("Cannot apply database migrations: %s", err)
	}
	log.Printf("Applied %d database migrations successfully", n)

	// Model configuration
	problemModel, err := NewSQLModel(db, ProblemModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", ProblemModelName, err)
	}

	algoModel, err := NewSQLModel(db, AlgoModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", AlgoModelName, err)
	}

	modelModel, err := NewSQLModel(db, ModelModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", ModelModelName, err)
	}

	dataModel, err := NewSQLModel(db, DataModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", DataModelName, err)
	}

	predictionModel, err := NewSQLModel(db, PredictionModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", PredictionModelName, err)
	}

	// Set BlobStore
	blobStore, err := SetBlobStore(*conf)
	if err != nil {
		log.Fatalf("Cannot set blobStore: %s", err)
	}

	api := &APIServer{
		Conf:            conf,
		BlobStore:       blobStore,
		ProblemModel:    problemModel,
		AlgoModel:       algoModel,
		ModelModel:      modelModel,
		DataModel:       dataModel,
		PredictionModel: predictionModel,
	}
	api.ConfigureRoutes(app, authentication)

	// Main server loop
	if conf.TLSOn() {
		app.ListenTLS(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port), conf.CertFile, conf.KeyFile)
	} else {
		app.Listen(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port))
	}
}

// misc routes
func (s *APIServer) index(c *iris.Context) {
	c.JSON(200, []string{
		RootRoute,
		HealthRoute,
		ProblemListRoute,
		ProblemRoute,
		ProblemBlobRoute,
		DataListRoute,
		DataRoute,
		DataBlobRoute,
		AlgoListRoute,
		AlgoRoute,
		AlgoBlobRoute,
		ModelListRoute,
		ModelRoute,
		ModelBlobRoute,
		PredictionRoute,
		PredictionBlobRoute,
	})
}

func (s *APIServer) health(c *iris.Context) {
	// TODO: check database and blob store connectivity here
	c.JSON(200, map[string]string{"status": "ok"})
}

// Generic blob routes and utilities
func (s *APIServer) getBlobKey(blobType string, blobID uuid.UUID) string {
	return fmt.Sprintf("%s/%s", blobType, blobID)
}

// Problem related routes
func (s *APIServer) getProblemList(c *iris.Context) {
	problems := make([]common.Problem, 0, 30)
	err := s.ProblemModel.List(&problems, 0, 30)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error retrieving problem list: %s", err)))
		return
	}

	c.JSON(200, map[string]interface{}{
		"page":   0,
		"length": len(problems),
		"items":  problems,
	})
}

func (s *APIServer) postProblem(c *iris.Context) {
	problem := common.NewProblem()
	statusCode, err := s.streamMultipartToStorage(s.ProblemModel, problem, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("[Error uploading problem] %s", err)))
		return
	}
	err = s.ProblemModel.Insert(problem)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error inserting problem %s in database: %s", problem.ID, err)))
	}
	c.JSON(201, problem)
}

func (s *APIServer) patchProblem(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}
	problem, err := s.getProblemInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving problem %s: %s", c.Param("uuid"), err)))
		return
	}
	statusCode, err := s.streamMultipartToStorage(s.ProblemModel, problem, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("[Error patching problem] %s", err)))
		return
	}
	if statusCode == 201 && problem.ID != id { // delete old blob if uuid and blob has changed
		err = s.BlobStore.Delete(s.getBlobKey(s.ProblemModel.GetModelName(), id))
	}
	err = s.ProblemModel.Update(problem, id)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error updating problem %s in database: %s", problem.ID, err)))
		return
	}
	c.JSON(200, problem)
}

func (s *APIServer) getProblemInstance(id uuid.UUID) (*common.Problem, error) {
	problem := common.Problem{}
	err := s.ProblemModel.GetOne(&problem, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving problem %s: %s", id, err))
	}
	return &problem, nil
}

func (s *APIServer) getProblem(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}

	problem, err := s.getProblemInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving problem %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(200, problem)
}

func (s *APIServer) getProblemBlob(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}
	_, err = s.getProblemInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving problem %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("problem", id, c)
}

// Algorithm related routes
func (s *APIServer) getAlgoList(c *iris.Context) {
	algos := make([]common.Algo, 0, 30)
	err := s.AlgoModel.List(&algos, 0, 30)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error retrieving algo list: %s", err)))
		return
	}

	c.JSON(200, map[string]interface{}{
		"page":   0,
		"length": len(algos),
		"items":  algos,
	})
}

func (s *APIServer) postAlgo(c *iris.Context) {
	algo := common.NewAlgo()
	statusCode, err := s.streamMultipartToStorage(s.AlgoModel, algo, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("[Error uploading algo] %s", err)))
		return
	}
	err = s.AlgoModel.Insert(algo)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error inserting algo %s in database: %s", algo.ID, err)))
	}
	c.JSON(201, algo)
}

func (s *APIServer) getAlgoInstance(id uuid.UUID) (*common.Algo, error) {
	algo := common.Algo{}
	err := s.AlgoModel.GetOne(&algo, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving algo %s: %s", id, err))
	}
	return &algo, nil
}

func (s *APIServer) getAlgo(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}

	algo, err := s.getAlgoInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving algo %s: %s", c.Param("uuid"), err)))
		return
	}
	c.JSON(200, algo)
}

func (s *APIServer) getAlgoBlob(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}

	_, err = s.getAlgoInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving algo %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("algo", id, c)
}

// Model related routes
func (s *APIServer) getModelList(c *iris.Context) {
	models := make([]common.Model, 0, 30)
	err := s.ModelModel.List(&models, 0, 30)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error retrieving model list: %s", err)))
		return
	}

	c.JSON(200, map[string]interface{}{
		"page":   0,
		"length": len(models),
		"items":  models,
	})
}

func (s *APIServer) postModel(c *iris.Context) {
	algoID, err := uuid.FromString(c.URLParam("algo"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse algo UUID %s: %s", algoID, err)))
		return
	}
	algo, err := s.getAlgoInstance(algoID)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error uploading model: algorithm %s not found: %s", c.URLParam("algo"), err)))
		return
	}

	modelID := uuid.Nil
	modelIDString := c.URLParam("uuid")
	if len(modelIDString) > 0 {
		modelID, err = uuid.FromString(modelIDString)
		if err != nil {
			c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse model UUID %s: %s", modelID, err)))
			return
		}
	}

	model := common.NewModel(modelID, algo)
	statusCode, err := s.streamBlobToStorage("model", model.ID, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("[Error uploading model] %s", err)))
		return
	}
	err = s.ModelModel.Insert(model)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error inserting model %s in database: %s", model.ID, err)))
	}
	c.JSON(201, model)
}

func (s *APIServer) getModelInstance(id uuid.UUID) (*common.Model, error) {
	model := common.Model{}
	err := s.ModelModel.GetOne(&model, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving model %s: %s", id, err))
	}
	return &model, nil
}

func (s *APIServer) getModel(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}

	model, err := s.getModelInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving model %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(200, model)
}

func (s *APIServer) getModelBlob(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}
	_, err = s.getModelInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving model %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("model", id, c)
}

// Data related routes
func (s *APIServer) getDataList(c *iris.Context) {
	datas := make([]common.Data, 0, 30)
	err := s.DataModel.List(&datas, 0, 30)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error retrieving data list: %s", err)))
		return
	}

	c.JSON(200, map[string]interface{}{
		"page":   0,
		"length": len(datas),
		"items":  datas,
	})
}

func (s *APIServer) postData(c *iris.Context) {
	data := common.NewData()
	statusCode, err := s.streamMultipartToStorage(s.DataModel, data, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("[Error uploading data] %s", err)))
		return
	}
	err = s.DataModel.Insert(data)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error inserting data %s in database: %s", data.ID, err)))
	}
	c.JSON(201, data)
}

func (s *APIServer) getDataInstance(id uuid.UUID) (*common.Data, error) {
	data := common.Data{}
	err := s.DataModel.GetOne(&data, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving data %s: %s", id, err))
	}
	return &data, nil
}

func (s *APIServer) getData(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}

	data, err := s.getDataInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving data %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(200, data)
}

func (s *APIServer) getDataBlob(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}

	_, err = s.getDataInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving data %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("data", id, c)
}

// Prediction related routes
func (s *APIServer) getPredictionList(c *iris.Context) {
	predictions := make([]common.Prediction, 0, 30)
	err := s.PredictionModel.List(&predictions, 0, 30)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error retrieving prediction list: %s", err)))
		return
	}

	c.JSON(200, map[string]interface{}{
		"page":   0,
		"length": len(predictions),
		"items":  predictions,
	})
}

func (s *APIServer) postPrediction(c *iris.Context) {
	prediction := common.NewPrediction()
	statusCode, err := s.streamMultipartToStorage(s.PredictionModel, prediction, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("[Error uploading prediction] %s", err)))
		return
	}
	err = s.PredictionModel.Insert(prediction)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error inserting prediction %s in database: %s", prediction.ID, err)))
	}
	c.JSON(201, prediction)
}

func (s *APIServer) getPredictionInstance(id uuid.UUID) (*common.Prediction, error) {
	prediction := common.Prediction{}
	err := s.PredictionModel.GetOne(&prediction, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving prediction %s: %s", id, err))
	}
	return &prediction, nil
}

func (s *APIServer) getPrediction(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}

	prediction, err := s.getPredictionInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving prediction %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(200, prediction)
}

func (s *APIServer) getPredictionBlob(c *iris.Context) {
	id, err := uuid.FromString(c.Param("uuid"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}
	_, err = s.getPredictionInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error retrieving prediction %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("prediction", id, c)
}

// SetBlobStore defines the blobstore type (local, fake, S3)
func SetBlobStore(conf StorageConfig) (common.BlobStore, error) {
	switch {
	case conf.BlobStore == "gc" && conf.GCBucket != "":
		log.Println("[GCBlobStore] Data stored on Google Cloud Storage")
		return common.NewGCBlobStore(conf.GCBucket)
	case conf.BlobStore == "s3" && conf.AWSBucket != "" && conf.AWSRegion != "":
		log.Println("[S3BlobStore] Data stored on Amazon S3")
		return common.NewS3BlobStore(conf.AWSBucket, conf.AWSRegion)
	case conf.BlobStore == "local":
		log.Println(fmt.Sprintf("[LocalBlobStore] Data is stored locally in directory: %s", conf.DataDir))
		return common.NewLocalBlobStore(conf.DataDir)
	case conf.BlobStore == "mock":
		log.Println("[MOCKBlobStore] Blobstore Mock used to 'store' data")
		return common.NewMOCKBlobStore(conf.DataDir)
	default:
		return nil, fmt.Errorf("Error setting BlobStore: Invalid configuration")
	}
}
