package main

import (
	"errors"
	"fmt"
	"github.com/MorpheoOrg/morpheo-go-packages/common"
	"github.com/satori/go.uuid"
	"gopkg.in/kataras/iris.v6"
	"io"
	// "log"
	"mime"
	"mime/multipart"
	"strconv"
	"strings"
)

// PostMultipartFields represents all the valid multipart fields for the routes
var PostMultipartFields = map[string][]string{
	ProblemListRoute: []string{"uuid", "name", "description", "size", "blob"},
	AlgoListRoute:    []string{"uuid", "name", "size", "blob"},
	// ModelListRoute:   []string{"uuid", "name", "size", "blob"},
	DataListRoute: []string{"uuid", "size", "blob"},
}

const (
	// StrFieldMaxLength is the max length for the multipart fields
	StrFieldMaxLength = 255 // in bytes
	intFieldMaxLength = 20  // in bytes
)

func readMultipartField(formName string, part io.ReadCloser, maxLength int) (string, error) {
	defer part.Close()
	buf := make([]byte, maxLength)
	offset := 0
	for {
		n, err := part.Read(buf[offset:])
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("Error reading %s: %s", formName, err)
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
		return "", fmt.Errorf("Buffer overflow reading %s (max length is %d in base 10): %s", formName, maxLength, err)
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

func (s *APIServer) streamMultipartToStorage(ResourceModel Model, resource common.Resource, c *iris.Context) (int, error) {
	mediaType, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		return 400, fmt.Errorf("Error parsing header \"Content-Type\": %s", err)
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		return 400, fmt.Errorf("Invalid media type: %s. Should be: multipart/form-data", mediaType)
	}

	reader := multipart.NewReader(c.Request.Body, params["boundary"])
	defer c.Request.Body.Close()

	var size int64
	formFields := make(map[string]interface{})
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 400, fmt.Errorf("Error parsing multipart data: %s", err)
		}

		switch formName := part.FormName(); formName {
		case "uuid":
			uuidStr, err := readMultipartField(formName, part, StrFieldMaxLength)
			if err != nil {
				return 400, fmt.Errorf("Error reading UUID %s", err)
			}
			id, err := uuid.FromString(uuidStr)
			if err != nil {
				return 400, fmt.Errorf("Error parsing UUID %s", err)
			}
			if err = ResourceModel.CheckUUIDNotUsed(id); err != nil {
				return 409, err
			}
			formFields["uuid"] = id
		case "description":
			formFields["description"], err = readMultipartField(formName, part, StrFieldMaxLength)
			if err != nil {
				return 400, fmt.Errorf("Error reading description: %s", err)
			}
		case "name":
			formFields["name"], err = readMultipartField(formName, part, StrFieldMaxLength)
			if err != nil {
				return 400, fmt.Errorf("Error reading Name: %s", err)
			}
		case "size":
			sizeStr, err := readMultipartField(formName, part, intFieldMaxLength)
			if err != nil {
				return 400, fmt.Errorf("Error reading size field: %s", err)
			}
			size, err = strconv.ParseInt(sizeStr, 10, 64)
			if err != nil {
				return 400, fmt.Errorf("Error parsing size field to integer: %s", err)
			}
		default:
			defer part.Close()
			if formName == "blob" {
				err := resource.FillResource(formFields)
				if err != nil {
					return 400, fmt.Errorf("Invalid form: %s. Make sure that each form field is sent before blob in the multipart/form", err)
				}
				if err = resource.Check(); err != nil {
					return 400, fmt.Errorf("Invalid form: %s. Make sure that each form field is sent before blob in the multipart/form", err)
				}
				if size == 0 {
					return 400, fmt.Errorf("Invalid form: 'Size' unset. Make sure that each form field is sent before blob in the multipart/form")
				}
				err = s.BlobStore.Put(s.getBlobKey(ResourceModel.GetModelName(), resource.GetUUID()), part, size)
				if err != nil {
					return 500, fmt.Errorf("Error writing blob content to storage: %s", err)
				}
				return 201, nil
			}
			return 400, fmt.Errorf("Unknown field \"%s\"", part.FormName())
		}
	}
	// If method is patch, fill resource and return 200 if patch is valid
	if c.Method() == "PATCH" {
		if err := resource.FillResource(formFields); err != nil {
			return 400, fmt.Errorf("Invalid form: %s. Make sure that each form field is sent before blob in the multipart/form", err)
		}
		if err = resource.Check(); err != nil {
			return 400, fmt.Errorf("Invalid form: %s. Make sure that each form field is sent before blob in the multipart/form", err)
		}
		if formFields["uuid"] != nil {
			id, err := uuid.FromString(c.Param("uuid"))
			if err != nil {
				return 400, fmt.Errorf("Impossible to parse UUID %s: %s", id, err)
			}
			err = s.BlobStore.Rename(s.getBlobKey(ResourceModel.GetModelName(), id), s.getBlobKey(ResourceModel.GetModelName(), resource.GetUUID()))
			if err != nil {
				return 500, fmt.Errorf("Error renaming blob on storage: %s", err)
			}
		}
		return 200, nil
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
