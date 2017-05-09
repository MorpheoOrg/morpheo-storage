package main

import (
	"fmt"
	"io"
	"strings"

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
	dataListRoute    = "/data"
	dataRoute        = "/data/:uuid"
	algoListRoute    = "/algo"
	algoRoute        = "/algo/:uuid"
)

type apiServer struct {
	conf      *StorageConfig
	blobStore common.BlobStore
	// objectStore common.ObjectStore
}

func (s *apiServer) configureRoutes(app *iris.Framework) {
	// Misc.
	app.Get(rootRoute, s.index)
	app.Get(healthRoute, s.health)

	// Problem
	app.Get(problemListRoute, s.getProblemList)
	app.Post(problemListRoute, s.postProblem)
	app.Get(problemRoute, s.getProblem)

	// Algo
	app.Get(algoListRoute, s.getAlgoList)
	app.Post(algoListRoute, s.postAlgo)
	app.Get(algoRoute, s.getAlgo)

	// Data
	app.Get(dataListRoute, s.getDataList)
	app.Post(dataListRoute, s.postData)
	app.Get(dataRoute, s.getData)
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

	// TODO: configure both blob storage and object storage
	api := &apiServer{
		conf: conf,
		blobStore: &common.LocalBlobStore{
			DataDir: "./data",
		},
	}
	api.configureRoutes(app)

	// Main server loop
	if conf.TLSOn() {
		app.ListenTLS(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port), conf.CertFile, conf.KeyFile)
	} else {
		app.Listen(fmt.Sprintf("%s:%d", conf.Hostname, conf.Port))
	}
}

func (s *apiServer) index(c *iris.Context) {
	c.JSON(iris.StatusOK, []string{rootRoute, healthRoute, problemRoute, algoRoute, dataRoute})
}

func (s *apiServer) health(c *iris.Context) {
	// TODO: check object store and blob store connectivity here
	c.JSON(iris.StatusOK, map[string]string{"status": "ok"})
}

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

func (s *apiServer) getBlobList(blobType string, c *iris.Context) {
	// TODO: database select
	c.JSON(iris.StatusCreated, map[string]string{"page": "0", "length": "0", "data": "TODO"})
}

func (s *apiServer) postBlob(blobType string, c *iris.Context) {
	id := uuid.NewV4()
	// TODO: database insert
	err := s.blobStore.Put(s.getBlobKey(blobType, id), c.Request.Body)
	defer c.Request.Body.Close()
	if err != nil {
		c.JSON(iris.StatusInternalServerError, common.NewAPIError(fmt.Sprintf("Error uploading %s: %s", blobType, err)))
		return
	}
	c.JSON(iris.StatusCreated, map[string]string{"status": fmt.Sprintf("%s created", blobType), "uuid": id.String()})
}

func (s *apiServer) getBlob(blobType string, c *iris.Context) {
	// TODO: database existence check
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

func (s *apiServer) getProblemList(c *iris.Context) {
	s.getBlobList("problem", c)
}

func (s *apiServer) postProblem(c *iris.Context) {
	s.postBlob("problem", c)
}

func (s *apiServer) getProblem(c *iris.Context) {
	s.getBlob("problem", c)
}

func (s *apiServer) getAlgoList(c *iris.Context) {
	s.getBlobList("algo", c)
}

func (s *apiServer) postAlgo(c *iris.Context) {
	s.postBlob("algo", c)
}

func (s *apiServer) getAlgo(c *iris.Context) {
	s.getBlob("algo", c)
}

func (s *apiServer) getDataList(c *iris.Context) {
	s.getBlobList("data", c)
}

func (s *apiServer) postData(c *iris.Context) {
	s.postBlob("data", c)
}

func (s *apiServer) getData(c *iris.Context) {
	s.getBlob("data", c)
}
