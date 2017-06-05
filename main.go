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
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
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
	RootRoute        = "/"
	HealthRoute      = "/health"
	ProblemListRoute = "/problem"
	ProblemRoute     = "/problem/:uuid"
	ProblemBlobRoute = "/problem/:uuid/blob"
	DataListRoute    = "/data"
	DataRoute        = "/data/:uuid"
	DataBlobRoute    = "/data/:uuid/blob"
	AlgoListRoute    = "/algo"
	AlgoRoute        = "/algo/:uuid"
	AlgoBlobRoute    = "/algo/:uuid/blob"
	ModelListRoute   = "/model"
	ModelRoute       = "/model/:uuid"
	ModelBlobRoute   = "/model/:uuid/blob"
)

const (
	strFieldMaxLength = 255 // in bytes
	intFieldMaxLength = 20  // in bytes
)

// APIServer represents the API configurations
type APIServer struct {
	Conf         *StorageConfig
	BlobStore    common.BlobStore
	ProblemModel Model
	AlgoModel    Model
	ModelModel   Model
	DataModel    Model
}

// ConfigureRoutes links the urls with the func and set authentication
func (s *APIServer) ConfigureRoutes(app *iris.Framework, authentication iris.HandlerFunc) {
	// Misc.
	app.Get(RootRoute, s.index)
	app.Get(HealthRoute, s.health)

	// Problem
	app.Get(ProblemListRoute, authentication, s.getProblemList)
	app.Post(ProblemListRoute, authentication, s.postProblem)
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

	// Set BlobStore
	blobStore, err := SetBlobStore(conf.DataDir, conf.AWSBucket, conf.AWSRegion)
	if err != nil {
		log.Fatalf("Cannot set blobStore: %s", err)
	}

	api := &APIServer{
		Conf:         conf,
		BlobStore:    blobStore,
		ProblemModel: problemModel,
		AlgoModel:    algoModel,
		ModelModel:   modelModel,
		DataModel:    dataModel,
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

func readMultipartField(formName string, part io.ReadCloser, fieldType string) (string, error) {
	defer part.Close()
	var maxLength int64
	if fieldType == "int" {
		maxLength = intFieldMaxLength
	} else {
		maxLength = strFieldMaxLength
	}
	buf := make([]byte, maxLength)
	offset := 0
	for {
		n, err := part.Read(buf[offset:])
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("Error reading %s %s: %s", formName, fieldType, err)
		}
		offset += n

		if err == io.EOF || offset == len(buf) {
			break
		}
	}

	// Buffer overflow test
	rest := make([]byte, 10)
	n, err := part.Read(rest)
	if err != io.EOF || n > 0 {
		return "", fmt.Errorf("Buffer overflow reading %s %s (max length is %d in base 10): %s", formName, fieldType, maxLength, err)
	}
	return string(buf[:offset]), nil
}

func (s *APIServer) streamBlobToStorage(blobType string, id uuid.UUID, c *iris.Context) (int, error) {
	size, err := strconv.ParseInt(c.Request.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return 400, fmt.Errorf("Error parsing header 'Content-Length': should be blob size in bytes. err: %s", err)
	}
	err = s.BlobStore.Put(s.getBlobKey(blobType, id), c.Request.Body, size)
	defer c.Request.Body.Close()
	if err != nil {
		return 500, err
	}
	return 201, nil
}

func (s *APIServer) streamBlobMultipartToStorage(blobType string, id uuid.UUID, mff *common.MultipartFormFields, c *iris.Context) (int, error) {
	mediaType, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		return 400, fmt.Errorf("Error parsing header \"Content-Type\": %s", err)
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		return 400, fmt.Errorf("Invalid media type: %s. Should be: multipart/form-data", mediaType)
	}

	reader := multipart.NewReader(c.Request.Body, params["boundary"])
	defer c.Request.Body.Close()

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 400, fmt.Errorf("Error parsing multipart data: %s", err)
		}

		switch formName := part.FormName(); formName {
		case "description":
			mff.Description, err = readMultipartField(formName, part, "string")
			if err != nil {
				return 400, fmt.Errorf("Error reading description: %s", err)
			}
		case "name":
			mff.Name, err = readMultipartField(formName, part, "string")
			if err != nil {
				return 400, fmt.Errorf("Error reading Name: %s", err)
			}
		case "size":
			sizeStr, err := readMultipartField(formName, part, "int")
			if err != nil {
				return 400, fmt.Errorf("Error reading size field: %s", err)
			}
			mff.Size, err = strconv.ParseInt(sizeStr, 10, 64)
			if err != nil {
				return 400, fmt.Errorf("Error parsing size field to integer: %s", err)
			}
		default:
			defer part.Close()
			if formName == "blob" {
				err := common.CheckFormFields(blobType, mff)
				if err != nil {
					return 400, fmt.Errorf("%s", err)
				}
				err = s.BlobStore.Put(s.getBlobKey(blobType, id), part, mff.Size)
				if err != nil {
					return 500, fmt.Errorf("Error writing blob content to storage: %s", err)
				}
				return 200, nil
			}

			return 400, fmt.Errorf("Unknown field \"%s\"", part.FormName())
		}
	}
	return 400, errors.New("Premature EOF while parsing request")
}

func (s *APIServer) streamBlobFromStorage(blobType string, blobID uuid.UUID, c *iris.Context) {
	blobReader, err := s.BlobStore.Get(s.getBlobKey(blobType, blobID))
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error retrieving %s %s: %s", blobType, blobID, err)))
		return
	}
	defer blobReader.Close()
	c.StreamWriter(func(w io.Writer) bool {
		_, err := io.Copy(w, blobReader)
		if err != nil {
			c.JSON(500, common.NewAPIError(fmt.Sprintf("Error reading %s %s: %s", blobType, blobID, err)))
			return false
		}
		return false
	})
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
	statusCode, err := s.streamBlobToStorage("problem", problem.ID, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("Error uploading problem - %s", err)))
		return
	}
	err = s.ProblemModel.Insert(problem)
	if err != nil {
		c.JSON(500, common.NewAPIError(fmt.Sprintf("Error inserting problem %s in database: %s", problem.ID, err)))
	}
	c.JSON(201, problem)
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
	mff := common.MultipartFormFields{}
	statusCode, err := s.streamBlobMultipartToStorage("algo", algo.ID, &mff, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("Error uploading algo - %s", err)))
		return
	}
	algo.Name = mff.Name
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
	id, err := uuid.FromString(c.URLParam("algo"))
	if err != nil {
		c.JSON(400, common.NewAPIError(fmt.Sprintf("Impossible to parse UUID %s: %s", id, err)))
		return
	}
	algo, err := s.getAlgoInstance(id)
	if err != nil {
		c.JSON(404, common.NewAPIError(fmt.Sprintf("Error uploading model: algorithm %s not found: %s", c.URLParam("algo"), err)))
		return
	}

	model := common.NewModel(algo)
	statusCode, err := s.streamBlobToStorage("model", model.ID, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("Error uploading model - %s", err)))
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
	err := s.AlgoModel.GetOne(&model, id)
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
	statusCode, err := s.streamBlobToStorage("data", data.ID, c)
	if err != nil {
		c.JSON(statusCode, common.NewAPIError(fmt.Sprintf("Error uploading data - %s", err)))
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

// SetBlobStore defines the blobstore type (local, fake, S3)
func SetBlobStore(dataDir string, awsBucket string, awsRegion string) (common.BlobStore, error) {
	switch {
	case awsBucket == "" || awsRegion == "":
		log.Println(fmt.Sprintf("[LocalBlobStore] Data is stored locally in directory: %s", dataDir))
		return common.NewLocalBlobStore(dataDir)
	case awsBucket == "fake" && awsRegion == "fake":
		return common.NewFakeBlobStore(dataDir)
	default:
		return common.NewS3BlobStore(awsBucket, awsRegion)
	}
}
