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
	"io"
	"log"
	"strconv"
	"strings"
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

	"github.com/MorpheoOrg/go-packages/common"
)

// Available HTTP routes
const (
	rootRoute        = "/"
	healthRoute      = "/health"
	problemListRoute = "/problem"
	problemRoute     = "/problem/:uuid"
	problemBlobRoute = "/problem/:uuid/blob"
	dataListRoute    = "/data"
	dataRoute        = "/data/:uuid"
	dataBlobRoute    = "/data/:uuid/blob"
	algoListRoute    = "/algo"
	algoRoute        = "/algo/:uuid"
	algoBlobRoute    = "/algo/:uuid/blob"
	modelListRoute   = "/model"
	modelRoute       = "/model/:uuid"
	modelBlobRoute   = "/model/:uuid/blob"
)

type apiServer struct {
	conf         *StorageConfig
	blobStore    common.BlobStore
	problemModel *Model
	algoModel    *Model
	modelModel   *Model
	dataModel    *Model
}

func (s *apiServer) configureRoutes(app *iris.Framework, authentication iris.HandlerFunc) {
	// Misc.
	app.Get(rootRoute, s.index)
	app.Get(healthRoute, s.health)

	// Problem
	app.Get(problemListRoute, authentication, s.getProblemList)
	app.Post(problemListRoute, authentication, s.postProblem)
	app.Get(problemRoute, authentication, s.getProblem)
	app.Get(problemBlobRoute, authentication, s.getProblemBlob)

	// Algo
	app.Get(algoListRoute, authentication, s.getAlgoList)
	app.Post(algoListRoute, authentication, s.postAlgo)
	app.Get(algoRoute, authentication, s.getAlgo)
	app.Get(algoBlobRoute, authentication, s.getAlgoBlob)

	// Model
	app.Get(modelListRoute, authentication, s.getModelList)
	app.Post(modelListRoute, authentication, s.postModel)
	app.Get(modelRoute, authentication, s.getModel)
	app.Get(modelBlobRoute, authentication, s.getModelBlob)

	// Data
	app.Get(dataListRoute, authentication, s.getDataList)
	app.Post(dataListRoute, authentication, s.postData)
	app.Get(dataRoute, authentication, s.getData)
	app.Get(dataBlobRoute, authentication, s.getDataBlob)
}

// Database migration routine
func runMigrations(db *sqlx.DB, migrationDir string, rollback bool) (int, error) {
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

func main() {
	// Parses CLI flags to generate the API config
	conf := NewStorageConfig()

	// Iris setup
	app := iris.New()
	app.Adapt(iris.DevLogger(), httprouter.New())

	// Iris authentication
	authConfig := basicauth.Config{
		Users:      map[string]string{conf.APIUser: conf.APIPassword},
		Realm:      "Authorization Required",
		ContextKey: "mycustomkey",
		Expires:    time.Duration(30) * time.Minute,
	}
	authentication := basicauth.New(authConfig)

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

	n, err := runMigrations(db, conf.DBMigrationsDir, conf.DBRollback)
	if err != nil {
		log.Fatalf("Cannot apply database migrations: %s", err)
	}
	log.Printf("Applied %d database migrations successfully", n)

	// Model configuration
	problemModel, err := NewModel(db, ProblemModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", ProblemModelName, err)
	}

	algoModel, err := NewModel(db, AlgoModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", AlgoModelName, err)
	}

	modelModel, err := NewModel(db, ModelModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", ModelModelName, err)
	}

	dataModel, err := NewModel(db, DataModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", DataModelName, err)
	}

	//Set BlobStore
	blobStore, err := setBlobStore(conf.DataDir, conf.AWSBucket, conf.AWSRegion)
	if err != nil {
		log.Fatalf("Cannot set blobStore: ", err)
	}

	api := &apiServer{
		conf:         conf,
		blobStore:    blobStore,
		problemModel: problemModel,
		algoModel:    algoModel,
		modelModel:   modelModel,
		dataModel:    dataModel,
	}
	api.configureRoutes(app, authentication)

	// Main server loop
	if conf.TLSOn() {
		app.ListenTLS(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port), conf.CertFile, conf.KeyFile)
	} else {
		app.Listen(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port))
	}
}

// misc routes
func (s *apiServer) index(c *iris.Context) {
	c.JSON(iris.StatusOK, []string{
		rootRoute,
		healthRoute,
		problemListRoute,
		problemRoute,
		problemBlobRoute,
		dataListRoute,
		dataRoute,
		dataBlobRoute,
		algoListRoute,
		algoRoute,
		algoBlobRoute,
		modelListRoute,
		modelRoute,
		modelBlobRoute,
	})
}

func (s *apiServer) health(c *iris.Context) {
	// TODO: check database and blob store connectivity here
	c.JSON(iris.StatusOK, map[string]string{"status": "ok"})
}

// Generic blob routes and utilities
func (s *apiServer) getBlobKey(blobType string, blobID uuid.UUID) string {
	return fmt.Sprintf("%s/%s", blobType, blobID)
}

func (s *apiServer) checkUUID(candidate string) (id uuid.UUID, err error) {
	if !strings.Contains(candidate, "-") {
		return uuid.Nil, fmt.Errorf("Invalid UUID: %s", candidate)
	}
	if id, err = uuid.FromString(candidate); err != nil {
		return uuid.Nil, fmt.Errorf("Invalid UUID: %s", candidate)
	}
	return id, nil
}

func (s *apiServer) streamBlobToStorage(blobType string, id uuid.UUID, c *iris.Context) error {
	size, err := strconv.ParseInt(c.Request.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}
	err = s.blobStore.Put(s.getBlobKey(blobType, id), c.Request.Body, size)
	defer c.Request.Body.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *apiServer) streamBlobFromStorage(blobType string, c *iris.Context) {
	blobID, err := s.checkUUID(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusBadRequest, common.NewAPIError(fmt.Sprintf("Error parsing %s uuid: %s", blobType, err)))
		return
	}

	blobReader, err := s.blobStore.Get(s.getBlobKey(blobType, blobID))
	if err != nil {
		c.JSON(iris.StatusBadRequest, common.NewAPIError(fmt.Sprintf("Error retrieving %s %s: %s", blobType, blobID, err)))
		return
	}
	defer blobReader.Close()
	c.StreamWriter(func(w io.Writer) bool {
		_, err := io.Copy(w, blobReader)
		if err != nil {
			c.JSON(iris.StatusBadRequest, common.NewAPIError(fmt.Sprintf("Error reading %s %s: %s", blobType, blobID, err)))
			return false
		}
		return false
	})
}

// Problem related routes
func (s *apiServer) getProblemList(c *iris.Context) {
	problems := make([]common.Problem, 0, 30)
	err := s.problemModel.List(&problems, 0, 30)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error retrieving problem list: %s", err)))
		return
	}

	c.JSON(iris.StatusOK, map[string]interface{}{
		"page":   0,
		"length": len(problems),
		"items":  problems,
	})
}

func (s *apiServer) postProblem(c *iris.Context) {
	problem := common.NewProblem()
	err := s.streamBlobToStorage("problem", problem.ID, c)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error uploading problem %s: %s", problem.ID, err)))
		return
	}
	err = s.problemModel.Insert(problem)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error inserting problem %s in database: %s", problem.ID, err)))
	}
	c.JSON(iris.StatusCreated, map[string]string{"status": "problem created", "uuid": problem.ID.String()})
}

func (s *apiServer) getProblemInstance(idString string) (*common.Problem, error) {
	id, err := uuid.FromString(idString)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", idString, err))
	}

	problem := common.Problem{}
	err = s.problemModel.GetOne(&problem, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving problem %s: %s", id, err))
	}
	return &problem, nil
}

func (s *apiServer) getProblem(c *iris.Context) {
	problem, err := s.getProblemInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving problem %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(iris.StatusOK, problem)
}

func (s *apiServer) getProblemBlob(c *iris.Context) {
	_, err := s.getProblemInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving problem %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("problem", c)
}

// Algorithm related routes
func (s *apiServer) getAlgoList(c *iris.Context) {
	algos := make([]common.Algo, 0, 30)
	err := s.algoModel.List(&algos, 0, 30)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error retrieving algo list: %s", err)))
		return
	}

	c.JSON(iris.StatusOK, map[string]interface{}{
		"page":   0,
		"length": len(algos),
		"items":  algos,
	})
}

func (s *apiServer) postAlgo(c *iris.Context) {
	algo := common.NewAlgo()
	err := s.streamBlobToStorage("algo", algo.ID, c)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error uploading algo %s: %s", algo.ID, err)))
		return
	}
	err = s.algoModel.Insert(algo)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error inserting algo %s in database: %s", algo.ID, err)))
	}
	c.JSON(iris.StatusCreated, map[string]string{"status": "algo created", "uuid": algo.ID.String()})
}

func (s *apiServer) getAlgoInstance(idString string) (*common.Algo, error) {
	id, err := uuid.FromString(idString)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", idString, err))
	}

	algo := common.Algo{}
	err = s.algoModel.GetOne(&algo, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving algo %s: %s", id, err))
	}
	return &algo, nil
}

func (s *apiServer) getAlgo(c *iris.Context) {
	algo, err := s.getAlgoInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving algo %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(iris.StatusOK, algo)
}

func (s *apiServer) getAlgoBlob(c *iris.Context) {
	_, err := s.getAlgoInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving algo %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("algo", c)
}

// Model related routes
func (s *apiServer) getModelList(c *iris.Context) {
	models := make([]common.Model, 0, 30)
	err := s.algoModel.List(&models, 0, 30)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error retrieving model list: %s", err)))
		return
	}

	c.JSON(iris.StatusOK, map[string]interface{}{
		"page":   0,
		"length": len(models),
		"items":  models,
	})
}

func (s *apiServer) postModel(c *iris.Context) {
	algo, err := s.getAlgoInstance(c.URLParam("algo"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error uploading model: algorithm %s not found: %s", c.Param("algo"), err)))
		return
	}

	model := common.NewModel(algo)
	err = s.streamBlobToStorage("model", model.ID, c)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error uploading model %s: %s", model.ID, err)))
		return
	}
	err = s.modelModel.Insert(model)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error inserting model %s in database: %s", model.ID, err)))
	}
	c.JSON(iris.StatusCreated, map[string]string{"status": "model created", "uuid": model.ID.String()})
}

func (s *apiServer) getModelInstance(idString string) (*common.Model, error) {
	id, err := uuid.FromString(idString)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", idString, err))
	}

	model := common.Model{}
	err = s.algoModel.GetOne(&model, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving model %s: %s", id, err))
	}
	return &model, nil
}

func (s *apiServer) getModel(c *iris.Context) {
	model, err := s.getModelInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving model %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(iris.StatusOK, model)
}

func (s *apiServer) getModelBlob(c *iris.Context) {
	_, err := s.getModelInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving model %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("model", c)
}

// Data related routes
func (s *apiServer) getDataList(c *iris.Context) {
	datas := make([]common.Data, 0, 30)
	err := s.dataModel.List(&datas, 0, 30)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error retrieving data list: %s", err)))
		return
	}

	c.JSON(iris.StatusOK, map[string]interface{}{
		"page":   0,
		"length": len(datas),
		"items":  datas,
	})
}

func (s *apiServer) postData(c *iris.Context) {
	data := common.NewData()
	err := s.streamBlobToStorage("data", data.ID, c)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error uploading data %s: %s", data.ID, err)))
		return
	}
	err = s.dataModel.Insert(data)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error inserting data %s in database: %s", data.ID, err)))
	}
	c.JSON(iris.StatusCreated, map[string]string{"status": "data created", "uuid": data.ID.String()})
}

func (s *apiServer) getDataInstance(idString string) (*common.Data, error) {
	id, err := uuid.FromString(idString)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", idString, err))
	}

	data := common.Data{}
	err = s.dataModel.GetOne(&data, id)
	if err != nil {
		return nil, common.NewAPIError(fmt.Sprintf("Error retrieving data %s: %s", id, err))
	}
	return &data, nil
}

func (s *apiServer) getData(c *iris.Context) {
	data, err := s.getDataInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving data %s: %s", c.Param("uuid"), err)))
		return
	}

	c.JSON(iris.StatusOK, data)
}

func (s *apiServer) getDataBlob(c *iris.Context) {
	_, err := s.getDataInstance(c.Param("uuid"))
	if err != nil {
		c.JSON(iris.StatusNotFound, common.NewAPIError(fmt.Sprintf("Error retrieving data %s: %s", c.Param("uuid"), err)))
		return
	}

	s.streamBlobFromStorage("data", c)
}

func setBlobStore(dataDir string, awsBucket string, awsRegion string) (common.BlobStore, error) {
	switch {
	case awsBucket == "" || awsRegion == "":
		log.Println(fmt.Sprintf("[LocalBlobStore] Data is stored locally in directory: %s", dataDir))
		return common.NewLocalBlobStore(dataDir)
	default:
		return common.NewS3BlobStore(awsBucket, awsRegion)
	}
}
