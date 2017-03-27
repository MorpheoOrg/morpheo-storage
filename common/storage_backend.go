package dccommon

import (
	"bytes"
	"fmt"
	"io"

	"github.com/satori/go.uuid"
)

type StorageBackend interface {
	GetData(id uuid.UUID) (dataReader io.Reader, err error)
	GetModel(id uuid.UUID) (modelReader io.Reader, err error)
}

type StorageAPIMock struct {
	evilDataUUID  string
	evilModelUUID string
}

// A mock for our tests
func NewStorageAPIMock() (s *StorageAPIMock) {
	return &StorageAPIMock{
		evilDataUUID:  "58bc25d9-712d-4a53-8e73-2d6ca4d837c2",
		evilModelUUID: "610e134a-ff45-4416-aaac-1b3398e4bba6",
	}
}

func (s *StorageAPIMock) GetData(id uuid.UUID) (dataReader io.Reader, err error) {
	if id.String() == s.evilDataUUID {
		return nil, fmt.Errorf("Data %s not found on storage", id)
	}

	return bytes.NewBufferString("datamock"), nil
}

func (s *StorageAPIMock) GetModel(id uuid.UUID) (dataReader io.Reader, err error) {
	if id.String() == s.evilModelUUID {
		return nil, fmt.Errorf("Data %s not found on storage", id)
	}

	return bytes.NewBufferString("modelmock"), nil
}
