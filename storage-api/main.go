package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/satori/go.uuid"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter" // <--- TODO or adaptors/gorillamux
	"gopkg.in/kataras/iris.v6/middleware/logger"

	"github.com/MorpheoOrg/go-morpheo/common"
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
)

type apiServer struct {
	conf         *StorageConfig
	blobStore    common.BlobStore
	problemModel *Model
	algoModel    *Model
	dataModel    *Model
}

func (s *apiServer) configureRoutes(app *iris.Framework) {
	// Misc.
	app.Get(rootRoute, s.index)
	app.Get(healthRoute, s.health)

	// Problem
	app.Get(problemListRoute, s.getProblemList)
	app.Post(problemListRoute, s.postProblem)
	app.Get(problemRoute, s.getProblem)
	app.Get(problemBlobRoute, s.getProblemBlob)

	// Algo
	app.Get(algoListRoute, s.getAlgoList)
	app.Post(algoListRoute, s.postAlgo)
	app.Get(algoRoute, s.getAlgo)
	app.Get(algoBlobRoute, s.getAlgoBlob)

	// Data
	app.Get(dataListRoute, s.getDataList)
	app.Post(dataListRoute, s.postData)
	app.Get(dataRoute, s.getData)
	app.Get(dataBlobRoute, s.getDataBlob)
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
	app.Adapt(iris.DevLogger())
	app.Adapt(httprouter.New())

	// Logging middleware configuration
	customLogger := logger.New(logger.Config{
		Status: true,
		IP:     true,
		Method: true,
		Path:   true,
	})
	app.Use(customLogger)

	// TODO: configure both blob storage and object storage with flags
	db, err := sqlx.Connect("postgres", "user=storage password=tooshort host=postgres port=5432 sslmode=disable dbname=morpheo")
	if err != nil {
		log.Fatalf("Cannot open connection to database: %s", err)
	}
	log.Println("Database connection ready")

	// Apply migrations (TODO: migrations dir in flag + rollback flag)
	n, err := runMigrations(db, "/migrations", false)
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

	dataModel, err := NewModel(db, DataModelName)
	if err != nil {
		log.Fatalf("Cannot create model %s: %s", DataModelName, err)
	}

	api := &apiServer{
		conf: conf,
		blobStore: &common.LocalBlobStore{
			DataDir: "./data",
		},
		problemModel: problemModel,
		algoModel:    algoModel,
		dataModel:    dataModel,
	}
	api.configureRoutes(app)

	// Main server loop
	if conf.TLSOn() {
		app.ListenTLS(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port), conf.CertFile, conf.KeyFile)
	} else {
		app.Listen(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port))
	}
}

// misc routes
func (s *apiServer) index(c *iris.Context) {
	c.JSON(iris.StatusOK, []string{rootRoute, healthRoute, problemRoute, algoRoute, dataRoute})
}

func (s *apiServer) health(c *iris.Context) {
	// TODO: check object store and blob store connectivity here
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
	err := s.blobStore.Put(s.getBlobKey(blobType, id), c.Request.Body)
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
		c.JSON(iris.StatusBadRequest, common.NewAPIError(fmt.Sprintf("Error retrieving %s %s:", blobType, blobID, err)))
		return
	}
	defer blobReader.Close()
	c.StreamWriter(func(w io.Writer) bool {
		_, err := io.Copy(w, blobReader)
		if err != nil {
			c.JSON(iris.StatusBadRequest, common.NewAPIError(fmt.Sprintf("Error reading %s %s:", blobType, blobID, err)))
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

func (s *apiServer) getAlgoList(c *iris.Context) {
	algos := make([]common.Algo, 0, 30)
	err := s.algoModel.List(&algos, 0, 30)
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error retrieving problem list: %s", err)))
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
