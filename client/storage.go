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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/MorpheoOrg/go-morpheo/common"
	"github.com/satori/go.uuid"
)

// Storage HTTP API routes
const (
	StorageProblemWorkflowRoute = "problem"
	StorageAlgoRoute            = "algo"
	StorageModelRoute           = "model"
	StorageDataRoute            = "data"
	BlobSuffix                  = "blob"
)

// Storage describes the storage service API
type Storage interface {
	GetData(id uuid.UUID) (data *common.Data, err error)
	GetAlgo(id uuid.UUID) (algo *common.Algo, err error)
	GetModel(id uuid.UUID) (model *common.Model, err error)
	GetProblemWorkflow(id uuid.UUID) (problem *common.Problem, err error)
	GetDataBlob(id uuid.UUID) (dataReader io.ReadCloser, err error)
	GetAlgoBlob(id uuid.UUID) (algoReader io.ReadCloser, err error)
	GetModelBlob(id uuid.UUID) (modelReader io.ReadCloser, err error)
	GetProblemWorkflowBlob(id uuid.UUID) (problemReader io.ReadCloser, err error)
	PostData(id uuid.UUID, dataReader io.Reader) error
	PostAlgo(id uuid.UUID, algoReader io.Reader) error
	PostModel(model *common.Model, algoReader io.Reader) error
	PostProblemWorkflow(id uuid.UUID, problemReader io.Reader) error
}

// StorageAPI is a wrapper around our storage HTTP API
type StorageAPI struct {
	Storage

	Hostname string
	Port     int
	User     string
	Password string
}

func (s *StorageAPI) getObjectBlob(prefix string, id uuid.UUID) (dataReader io.ReadCloser, err error) {
	url := fmt.Sprintf("http://%s:%d/%s/%s/%s", s.Hostname, s.Port, prefix, id, BlobSuffix)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[storage-api] Error building GET request against %s: %s", url, err)
	}
	req.SetBasicAuth(s.User, s.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[storage-api] Error performing GET request against %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[storage-api] Bad status code (%s) performing GET request against %s", resp.Status, url)
	}

	return resp.Body, nil
}

func (s *StorageAPI) getAndParseJSONObject(objectRoute string, objectID uuid.UUID, dest interface{}) error {
	url := fmt.Sprintf("http://%s:%d/%s/%s", s.Hostname, s.Port, objectRoute, objectID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("[storage-api] Error building GET request against %s: %s", url, err)
	}
	req.SetBasicAuth(s.User, s.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("[storage-api] Error performing GET request against %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[storage-api] Bad status code (%s) performing GET request against %s", url, err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(dest)
	if err != nil {
		return fmt.Errorf("[storage-api] Error unmarshaling object retrieved from %s: %s", url, err)
	}

	return nil
}

func (s *StorageAPI) postObjectBlob(prefix string, id uuid.UUID, dataReader io.Reader) error {
	url := fmt.Sprintf("http://%s:%d/%s", s.Hostname, s.Port, prefix)

	req, err := http.NewRequest(http.MethodPost, url, dataReader)
	if err != nil {
		return fmt.Errorf("[storage-api] Error building streaming POST request against %s: %s", url, err)
	}
	req.SetBasicAuth(s.User, s.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("[storage-api] Error performing streaming POST request against %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("[storage-api] Bad status code (%s) performing streaming POST request against %s", resp.Status, url)
	}

	return nil
}

// GetProblemWorkflow returns a ProblemWorkflow's metadata
func (s *StorageAPI) GetProblemWorkflow(id uuid.UUID) (problem *common.Problem, err error) {
	problem = &common.Problem{}
	err = s.getAndParseJSONObject(StorageProblemWorkflowRoute, id, problem)
	return problem, err
}

// GetAlgo returns an Algo's metadata
func (s *StorageAPI) GetAlgo(id uuid.UUID) (algo *common.Algo, err error) {
	algo = &common.Algo{}
	err = s.getAndParseJSONObject(StorageAlgoRoute, id, algo)
	return algo, err
}

// GetModel returns a Model's metadata
func (s *StorageAPI) GetModel(id uuid.UUID) (model *common.Model, err error) {
	model = &common.Model{}
	err = s.getAndParseJSONObject(StorageModelRoute, id, model)
	return model, err
}

// GetData returns a dataset's metadata
func (s *StorageAPI) GetData(id uuid.UUID) (data *common.Data, err error) {
	data = &common.Data{}
	err = s.getAndParseJSONObject(StorageDataRoute, id, data)
	return data, err
}

// GetProblemWorkflowBlob returns an io.ReadCloser to a problem workflow image
//
// Note that it is up to the caller to call Close() on the returned io.ReadCloser
func (s *StorageAPI) GetProblemWorkflowBlob(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObjectBlob(StorageProblemWorkflowRoute, id)
}

// GetAlgoBlob returns an io.ReadCloser to a algo image (a .tar.gz file of the image's build
// context)
//
// Note that it is up to the caller to call Close() on the returned io.ReadCloser
func (s *StorageAPI) GetAlgoBlob(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObjectBlob(StorageAlgoRoute, id)
}

// GetModelBlob returns an io.ReadCloser to a model (a .tar.gz of the model volume)
//
// Note that it is up to the caller to call Close() on the returned io.ReadCloser
func (s *StorageAPI) GetModelBlob(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObjectBlob(StorageModelRoute, id)
}

// GetDataBlob returns an io.ReadCloser to a data image (a .tar.gz file of the dataset)
//
// Note that it is up to the caller to call Close() on the returned io.ReadCloser
func (s *StorageAPI) GetDataBlob(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObjectBlob(StorageDataRoute, id)
}

// PostProblemWorkflow returns an io.ReadCloser to a problem workflow image (a .tar.gz file on the
// image's build context)
func (s *StorageAPI) PostProblemWorkflow(id uuid.UUID, problemReader io.Reader) error {
	return s.postObjectBlob(StorageProblemWorkflowRoute, id, problemReader)
}

// PostAlgo returns an io.ReadCloser to a algo image
func (s *StorageAPI) PostAlgo(id uuid.UUID, algoReader io.Reader) error {
	return s.postObjectBlob(StorageAlgoRoute, id, algoReader)
}

// PostModel returns an io.ReadCloser to a model
func (s *StorageAPI) PostModel(model *common.Model, modelReader io.Reader) error {
	// Check for associated Algo existence
	if _, err := s.GetAlgo(model.Algo); err != nil {
		return fmt.Errorf("Algorithm %s associated to posted model wasn't found", model.Algo)
	}

	return s.postObjectBlob(fmt.Sprintf("%s?algo=%s", StorageModelRoute, model.Algo), model.ID, modelReader)
}

// PostData returns an io.ReadCloser to a data image
func (s *StorageAPI) PostData(id uuid.UUID, dataReader io.Reader) error {
	return s.postObjectBlob(StorageDataRoute, id, dataReader)
}

// StorageAPIMock is a mock of the storage API (for tests & local dev. purposes)
type StorageAPIMock struct {
	Storage

	evilDataUUID    string
	evilAlgoUUID    string
	evilProblemUUID string
}

// NewStorageAPIMock instantiates our mock of the storage API
func NewStorageAPIMock() (s *StorageAPIMock) {
	return &StorageAPIMock{
		evilDataUUID:    "58bc25d9-712d-4a53-8e73-2d6ca4d837c2",
		evilAlgoUUID:    "610e134a-ff45-4416-aaac-1b3398e4bba6",
		evilProblemUUID: "8f6563df-941d-4967-b517-f45169834741",
	}
}

// GetData returns fake data (the same, no matter the UUID)
func (s *StorageAPIMock) GetData(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	if id.String() == s.evilDataUUID {
		return nil, fmt.Errorf("Data %s not found on storage", id)
	}

	return ioutil.NopCloser(bytes.NewBufferString("datamock")), nil
}

// GetAlgo returns a fake algo, no matter the UUID
func (s *StorageAPIMock) GetAlgo(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	if id.String() == s.evilAlgoUUID {
		return nil, fmt.Errorf("Algo %s not found on storage", id)
	}

	return ioutil.NopCloser(bytes.NewBufferString("algomock")), nil
}

// GetProblemWorkflow returns a fake algo, no matter the UUID
func (s *StorageAPIMock) GetProblemWorkflow(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	if id.String() == s.evilAlgoUUID {
		return nil, fmt.Errorf("Problem workflow %s not found on storage", id)
	}

	return ioutil.NopCloser(bytes.NewBufferString("problemmock")), nil
}

// PostData forwards the given reader data bytes... to /dev/null AHAHAHAH !
func (s *StorageAPIMock) PostData(id uuid.UUID, dataReader io.Reader) error {
	_, err := io.Copy(ioutil.Discard, dataReader)
	return err
}

// PostAlgo sends an algorithm... to oblivion
func (s *StorageAPIMock) PostAlgo(id uuid.UUID, algoReader io.Reader) error {
	_, err := io.Copy(ioutil.Discard, algoReader)
	return err
}

// PostProblemWorkflow fills the universe with one more problem, but the universe doesn't care
func (s *StorageAPIMock) PostProblemWorkflow(id uuid.UUID, problemReader io.Reader) error {
	_, err := io.Copy(ioutil.Discard, problemReader)
	return err
}
