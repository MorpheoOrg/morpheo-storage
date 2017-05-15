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
	StorageDataRoute            = "data"
	BlobSuffix                  = "blob"
)

// Storage describes the storage service API
type Storage interface {
	GetData(id uuid.UUID) (data *common.Data, err error)
	GetAlgo(id uuid.UUID) (algo *common.Algo, err error)
	GetProblemWorkflow(id uuid.UUID) (problem *common.Problem, err error)
	GetDataBlob(id uuid.UUID) (dataReader io.ReadCloser, err error)
	GetAlgoBlob(id uuid.UUID) (algoReader io.ReadCloser, err error)
	GetProblemWorkflowBlob(id uuid.UUID) (problemReader io.ReadCloser, err error)
	PostData(id uuid.UUID, dataReader io.Reader) error
	PostAlgo(id uuid.UUID, algoReader io.Reader) error
	PostProblemWorkflow(id uuid.UUID, problemReader io.Reader) error
}

// StorageAPI is a wrapper around our storage HTTP API
type StorageAPI struct {
	Storage

	Hostname string
	Port     int
}

func (s *StorageAPI) getObjectBlob(prefix string, id uuid.UUID) (dataReader io.ReadCloser, err error) {
	url := fmt.Sprintf("http://%s:%d/%s/%s/%s", s.Hostname, s.Port, prefix, id, BlobSuffix)
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

func (s *StorageAPI) getAndParseJSONObject(objectRoute string, objectID uuid.UUID, dest interface{}) error {
	url := fmt.Sprintf("http://%s:%d/%s/%s", s.Hostname, s.Port, objectRoute, objectID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("[storage-api] Error building GET request against %s: %s", url, err)
	}

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
	err = s.getAndParseJSONObject(StorageProblemWorkflowRoute, id, problem)
	return
}

// GetAlgo returns an Algo's metadata
func (s *StorageAPI) GetAlgo(id uuid.UUID) (algo *common.Algo, err error) {
	err = s.getAndParseJSONObject(StorageAlgoRoute, id, algo)
	return
}

// GetData returns a dataset's metadata
func (s *StorageAPI) GetData(id uuid.UUID) (data *common.Data, err error) {
	err = s.getAndParseJSONObject(StorageDataRoute, id, data)
	return
}

// GetProblemWorkflowBlob returns an io.ReadCloser to a problem workflow image
func (s *StorageAPI) GetProblemWorkflowBlob(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObjectBlob(StorageProblemWorkflowRoute, id)
}

// GetAlgoBlob returns an io.ReadCloser to a algo image (a .tar.gz file of the image's build
// context)
func (s *StorageAPI) GetAlgoBlob(id uuid.UUID) (dataReader io.ReadCloser, err error) {
	return s.getObjectBlob(StorageAlgoRoute, id)
}

// GetDataBlob returns an io.ReadCloser to a data image (a .tar.gz file of the dataset)
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
