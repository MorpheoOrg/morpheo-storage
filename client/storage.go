package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/satori/go.uuid"
)

// Storage HTTP API routes
const (
	StorageProblemWorkflowRoute = "/problem"
	StorageModelRoute           = "/algo"
	StorageDataRoute            = "/data"
)

// Storage describes the storage service API
type Storage interface {
	GetData(id uuid.UUID) (dataReader io.ReadCloser, err error)
	GetModel(id uuid.UUID) (modelReader io.ReadCloser, err error)
	GetProblemWorkflow(id uuid.UUID) (problemReader io.ReadCloser, err error)
	PostData(id uuid.UUID, dataReader io.Reader) error
	PostModel(id uuid.UUID, modelReader io.Reader) error
	PostProblemWorkflow(id uuid.UUID, problemReader io.Reader) error
}

// StorageAPI is a wrapper around our storage HTTP API
type StorageAPI struct {
	Storage

	Hostname string
	Port     int
}

func (s *StorageAPI) getObject(prefix string, id uuid.UUID) (dataReader io.ReadCloser, err error) {
	url := fmt.Sprintf("http://%s:%d%s", s.Hostname, s.Port, prefix)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[storage-api] Error building GET request against %s: %s", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[storage-api] Error performing GET request against %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[storage-api] Bad status code (%s) performing GET request against %s", resp.Status, url)
	}

	return resp.Body, nil
}

func (s *StorageAPI) streamObject(prefix string, id uuid.UUID, dataReader io.Reader) error {
	url := fmt.Sprintf("http://%s:%d%s", s.Hostname, s.Port, prefix)

	req, err := http.NewRequest(http.MethodPost, url, dataReader)
	if err != nil {
		return fmt.Errorf("[storage-api] Error building streaming POST request against %s: %s", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("[storage-api] Error performing streaming POST request against %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("[storage-api] Bad status code (%s) performing streaming POST request against %s", resp.Status, url)
	}

	return nil
}

// GetProblemWorkflow returns an io.ReadCloser to a problem workflow image
func (s *StorageAPI) GetProblemWorkflow(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObject(StorageProblemWorkflowRoute, id)
}

// GetModel returns an io.ReadCloser to a model image
func (s *StorageAPI) GetModel(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObject(StorageModelRoute, id)
}

// GetData returns an io.ReadCloser to a data image
func (s *StorageAPI) GetData(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObject(StorageDataRoute, id)
}

// PostProblemWorkflow returns an io.ReadCloser to a problem workflow image
func (s *StorageAPI) PostProblemWorkflow(id uuid.UUID, problemReader io.Reader) error {
	return s.streamObject(StorageProblemWorkflowRoute, id, problemReader)
}

// PostModel returns an io.ReadCloser to a model image
func (s *StorageAPI) PostModel(id uuid.UUID, modelReader io.Reader) error {
	return s.streamObject(StorageModelRoute, id, modelReader)
}

// PostData returns an io.ReadCloser to a data image
func (s *StorageAPI) PostData(id uuid.UUID, dataReader io.Reader) error {
	return s.streamObject(StorageDataRoute, id, dataReader)
}

// StorageAPIMock is a mock of the storage API (for tests & local dev. purposes)
type StorageAPIMock struct {
	Storage

	evilDataUUID    string
	evilModelUUID   string
	evilProblemUUID string
}

// NewStorageAPIMock instantiates our mock of the storage API
func NewStorageAPIMock() (s *StorageAPIMock) {
	return &StorageAPIMock{
		evilDataUUID:    "58bc25d9-712d-4a53-8e73-2d6ca4d837c2",
		evilModelUUID:   "610e134a-ff45-4416-aaac-1b3398e4bba6",
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

// GetModel returns a fake model, no matter the UUID
func (s *StorageAPIMock) GetModel(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	if id.String() == s.evilModelUUID {
		return nil, fmt.Errorf("Model %s not found on storage", id)
	}

	return ioutil.NopCloser(bytes.NewBufferString("modelmock")), nil
}

// GetProblemWorkflow returns a fake model, no matter the UUID
func (s *StorageAPIMock) GetProblemWorkflow(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	if id.String() == s.evilModelUUID {
		return nil, fmt.Errorf("Problem workflow %s not found on storage", id)
	}

	return ioutil.NopCloser(bytes.NewBufferString("problemmock")), nil
}

// PostData forwards the given reader data bytes... to /dev/null AHAHAHAH !
func (s *StorageAPIMock) PostData(id uuid.UUID, dataReader io.Reader) error {
	_, err := io.Copy(ioutil.Discard, dataReader)
	return err
}

// PostModel sends a model... to oblivion
func (s *StorageAPIMock) PostModel(id uuid.UUID, modelReader io.Reader) error {
	_, err := io.Copy(ioutil.Discard, modelReader)
	return err
}

// PostProblemWorkflow fills the universe with one more problem, but the universe doesn't care
func (s *StorageAPIMock) PostProblemWorkflow(id uuid.UUID, problemReader io.Reader) error {
	_, err := io.Copy(ioutil.Discard, problemReader)
	return err
}
