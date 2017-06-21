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

package main_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/MorpheoOrg/go-packages/common"
	. "github.com/MorpheoOrg/storage"
	"github.com/satori/go.uuid"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/httptest"
)

var (
	app          *iris.Framework
	objectRoutes = []string{
		ProblemListRoute, ProblemRoute, ProblemBlobRoute,
		DataListRoute, DataRoute, DataBlobRoute,
		AlgoListRoute, AlgoRoute, AlgoBlobRoute,
		ModelListRoute, ModelRoute, ModelBlobRoute,
	}
	listObjectRoutes          = []string{DataListRoute, ProblemListRoute, AlgoListRoute, ModelListRoute}
	postObjectBinaryRoutes    = []string{DataListRoute, ProblemListRoute}
	postObjectMultipartRoutes = []string{AlgoListRoute}
	randomUUID                string
)

func TestMain(m *testing.M) {
	fmt.Printf("Test starting bitch!\n")
	app = setTestApp()
	randomUUID = uuid.NewV4().String()
	os.Exit(m.Run())
}

// Test valid Root request returns Success
func TestRootRoute(t *testing.T) {
	e := httptest.New(app, t)
	e.GET(RootRoute).Expect().Status(200)
}

// Test valid Health request returns Success
func TestHealthRoute(t *testing.T) {
	e := httptest.New(app, t)
	e.GET(HealthRoute).Expect().Status(200).JSON().Equal(map[string]interface{}{"status": "ok"})
}

func TestRouteAuthentication(t *testing.T) {
	e := httptest.New(app, t)

	// Test routes access unauthorized without authentication
	for _, route := range objectRoutes {
		e.GET(route).Expect().Status(401)
	}

	// Test routes access unauthorized with wrong authentication
	e.GET(objectRoutes[0]).WithBasicAuth("invalid", "invalid").Expect().Status(401)
}

func TestGetListObject(t *testing.T) {
	e := httptest.New(app, t)

	for _, url := range listObjectRoutes {
		t.Logf(url)

		// Test valid request returns Success
		e.GET(url).WithBasicAuth("u", "p").Expect().Status(200)
	}
}

func TestGetObject(t *testing.T) {

	for _, url := range listObjectRoutes {
		t.Logf(url)

		// Test valid request returns Success
		e := httptest.New(app, t)
		e.GET(url+"/"+randomUUID).WithBasicAuth("u", "p").Expect().Status(200)

		// Test invalid uuid returns BadRequest
		e.GET(url+"/666devil").WithBasicAuth("u", "p").Expect().Status(400).Body().Match("(.*)Impossible to parse UUID(.*)")

		// Test uuid not in db returns NotFound
		e.GET(url+"/"+DevilMockUUID).WithBasicAuth("u", "p").Expect().Status(404).Body().Match("{(.*)sql: no rows in result set\"}")
	}
}

func TestGetObjectBlob(t *testing.T) {
	e := httptest.New(app, t)

	for _, url := range listObjectRoutes {
		t.Logf(url)

		// Test valid request returns Success
		e.GET(url+"/"+randomUUID+"/blob").WithBasicAuth("u", "p").Expect().Status(200)

		// Test invalid uuid returns BadRequest and error 'Impossible to parse'
		e.GET(url+"/666devil/blob").WithBasicAuth("u", "p").Expect().Status(400).Body().Match("(.*)Impossible to parse UUID(.*)")

		// Test uuid not in db returns NotFound
		e.GET(url+"/"+DevilMockUUID+"/blob").WithBasicAuth("u", "p").Expect().Status(404).Body().Match("{(.*)sql: no rows in result set\"}")

		// test download failed returns internalServerError
		e.GET(url+"/"+common.ViciousDevilUUID+"/blob").WithBasicAuth("u", "p").Expect().Status(500)
	}
}

func TestPostObjectBinary(t *testing.T) {
	e := httptest.New(app, t)

	for _, url := range postObjectBinaryRoutes {
		t.Logf(url)

		// Test valid request returns StatusCreated
		e.POST(url).WithBasicAuth("u", "p").WithHeader("Content-Length", "666").WithBytes([]byte("fakefilecontent")).Expect().Status(201)

		// Test request without Content-Length Header returns BadRequest
		e.POST(url).WithBasicAuth("u", "p").WithBytes([]byte("fakefilecontent")).Expect().Status(400).Body().Match("(.*)Error parsing header 'Content-Length'(.*)")

		// Test request with invalid Content-Length Header returns BadRequest
		e.POST(url).WithBasicAuth("u", "p").WithHeader("Content-Length", "James Bond").WithBytes([]byte("fakefilecontent")).Expect().Status(400)

		// Test failed file upload returns InternalServerError
		e.POST(url).WithBasicAuth("u", "p").WithHeader("Content-Length", strconv.Itoa(common.NaughtySize)).WithBytes([]byte("fakefilecontent")).Expect().Status(500).Body().Match("(.*)What a naughty size(.*)")

		// TODO: Check that if size field does not fit file size, cancel streamming...
	}
}

func TestPostObjectMultipart(t *testing.T) {
	e := httptest.New(app, t)

	// Test request without Content-Type header returns BadRequest
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").Expect().Status(400)

	// Test request with invalid Content-Type header returns BadRequest
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithHeader("Content-Type", "invalid").Expect().Status(400)
}

func TestPostModel(t *testing.T) {
	e := httptest.New(app, t)

	// Test valid request returns StatusCreated
	e.POST(ModelListRoute).WithQuery("algo", randomUUID).WithBasicAuth("u", "p").WithHeader("Content-Length", "15").WithBytes([]byte("fakefilecontent")).Expect().Status(201)

	// Test request with unvalid algo uuid returns BadRequest
	e.POST(ModelListRoute).WithQuery("algo", "7-Batman").WithBasicAuth("u", "p").WithHeader("Content-Length", "15").WithBytes([]byte("fakefilecontent")).Expect().Status(400)

	// Test request with unexistant algo uuid returns NotFound
	e.POST(ModelListRoute).WithQuery("algo", DevilMockUUID).WithBasicAuth("u", "p").WithHeader("Content-Length", "15").WithBytes([]byte("fakefilecontent")).Expect().Status(404).Body().Match(`{\"error\":\"Error uploading model: algorithm (.+) not found: Error retrieving algo (.+): (.*)\"}`)

	// Test failed file upload returns InternalServerError
	e.POST(ModelListRoute).WithQuery("algo", randomUUID).WithBasicAuth("u", "p").WithHeader("Content-Length", strconv.Itoa(common.NaughtySize)).WithBytes([]byte("fakefilecontent")).Expect().Status(500).Body().Match("(.*)What a naughty size(.*)")
}

func TestPostAlgoMultipart(t *testing.T) {
	e := httptest.New(app, t)

	// Test valid request returns StatusCreated
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "test").WithFormField("size", "10").WithFile("blob", "README.md").Expect().Status(201)

	// Test one empty form fields returns BadRequest
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "").WithFormField("size", "10").WithFile("blob", "README.md").Expect().Status(400)
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "test").WithFormField("size", "").WithFile("blob", "README.md").Expect().Status(400)

	// Test field blob not at the end returns BadRequest
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "test").WithFile("blob", "README.md").WithFormField("size", "").Expect().Status(400)
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("size", "3").WithFile("blob", "README.md").WithFormField("name", "test").Expect().Status(400)
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFile("blob", "README.md").WithFormField("size", "3").WithFormField("name", "test").Expect().Status(400)

	// Test request with description field returns BadRequest
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "test").WithFormField("description", "awesome").WithFormField("size", "10").WithFile("blob", "README.md").Expect().Status(400)

	// Test request with invalid field returns BadRequest
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "test").WithFormField("invalid", "test").WithFormField("size", "10").WithFile("blob", "README.md").Expect().Status(400)

	// Test request with invalid size field returns BadRequest
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "test").WithFormField("size", "invalid").WithFile("blob", "README.md").Expect().Status(400)

	// Test failed file upload returns InternalServerError
	e.POST(AlgoListRoute).WithBasicAuth("u", "p").WithMultipart().WithFormField("name", "test").WithFormField("size", common.NaughtySize).WithFile("blob", "README.md").Expect().Status(500)
}

// setTestApp set up the Iris App for testing
func setTestApp() *iris.Framework {
	_ = NewStorageConfig()
	app := iris.New()
	app.Adapt(iris.DevLogger(), httprouter.New())
	auth := SetAuthentication("u", "p")

	// Set models configuration
	problemModel, _ := NewMockedModel(ProblemModelName)
	algoModel, _ := NewMockedModel(AlgoModelName)
	modelModel, _ := NewMockedModel(ModelModelName)
	dataModel, _ := NewMockedModel(DataModelName)

	// set Blobstore
	blobStore, _ := SetBlobStore("fake", "fake", "fake")

	api := &APIServer{
		BlobStore:    blobStore,
		ProblemModel: problemModel,
		AlgoModel:    algoModel,
		ModelModel:   modelModel,
		DataModel:    dataModel,
	}
	api.ConfigureRoutes(app, auth)
	return app
}
