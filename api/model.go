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
	"github.com/fatih/structs"
	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"
	"reflect"
)

// Model (and SQL table) names
const (
	ProblemModelName    = "problem"
	AlgoModelName       = "algo"
	DataModelName       = "data"
	ModelModelName      = "model"
	PredictionModelName = "prediction"
	migrationTable      = "storage_migrations"
	DevilMockUUID       = "c54e361e-18db-48dd-aa71-96f28a1af892"
	ProblemMockUUIDStr  = "e42a31bb-a97b-47ff-81cf-ffdd7c5ddd08"
)

var (
	// SQL statements
	insertStatements = map[string]string{
		"problem":    `INSERT INTO problem (uuid, timestamp_upload, name, description ) VALUES (:uuid, :timestamp_upload, :name, :description)`,
		"algo":       `INSERT INTO algo (uuid, timestamp_upload, name) VALUES (:uuid, :timestamp_upload, :name)`,
		"model":      `INSERT INTO model (uuid, algo, timestamp_upload) VALUES (:uuid, :algo, :timestamp_upload)`,
		"data":       `INSERT INTO data (uuid, timestamp_upload) VALUES (:uuid, :timestamp_upload)`,
		"prediction": `INSERT INTO prediction (uuid, timestamp_upload) VALUES (:uuid, :timestamp_upload)`,
	}
	selectTemplates = map[string]string{
		"problem":    "SELECT * FROM problem ORDER BY timestamp_upload DESC LIMIT %d OFFSET %d",
		"algo":       "SELECT * FROM algo ORDER BY timestamp_upload DESC LIMIT %d OFFSET %d",
		"model":      "SELECT * FROM model ORDER BY timestamp_upload DESC LIMIT %d OFFSET %d",
		"data":       "SELECT * FROM data ORDER BY timestamp_upload DESC LIMIT %d OFFSET %d",
		"prediction": "SELECT * FROM prediction ORDER BY timestamp_upload DESC LIMIT %d OFFSET %d",
	}
	getOneStatements = map[string]string{
		"problem":    `SELECT * FROM problem WHERE uuid=$1 LIMIT 1`,
		"algo":       `SELECT * FROM algo WHERE uuid=$1 LIMIT 1`,
		"model":      `SELECT * FROM model WHERE uuid=$1 LIMIT 1`,
		"data":       `SELECT * FROM data WHERE uuid=$1 LIMIT 1`,
		"prediction": `SELECT * FROM prediction WHERE uuid=$1 LIMIT 1`,
	}
	updateStatements = map[string]string{
		"problem": `UPDATE problem SET uuid=:ID, timestamp_upload=:TimestampUpload, name=:Name, description=:Description WHERE uuid=:prev_uuid`,
	}

	// Valid model names
	modelNames = map[string]struct{}{
		ProblemModelName:    struct{}{},
		AlgoModelName:       struct{}{},
		DataModelName:       struct{}{},
		ModelModelName:      struct{}{},
		PredictionModelName: struct{}{},
	}
)

// Model contains methods to interact with models stored in base
type Model interface {
	Insert(instance interface{}) error
	List(instanceList interface{}, page, pageSize int) error
	GetOne(instance interface{}, id uuid.UUID) error
	Update(instance interface{}, id uuid.UUID) error
	CheckUUIDNotUsed(id uuid.UUID) error
	GetModelName() string
}

// SQLModel interacts with a postgreSQL database
type SQLModel struct {
	*sqlx.DB

	name string
}

// NewSQLModel creates a Model instance, bound to a given database
func NewSQLModel(db *sqlx.DB, name string) (*SQLModel, error) {
	if _, ok := modelNames[name]; !ok {
		return nil, fmt.Errorf("Unknown model %s", name)
	}
	return &SQLModel{db, name}, nil
}

// Insert inserts a given model instance in base
func (m *SQLModel) Insert(instance interface{}) error {
	if insertStatement, ok := insertStatements[m.name]; ok {
		if _, err := m.NamedExec(insertStatement, instance); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("[model] No insert statement found for model %s", m.name)
	}
	return nil
}

// List lists all model instances in base, pagination included
func (m *SQLModel) List(instanceList interface{}, page, pageSize int) error {
	if selectTemplate, ok := selectTemplates[m.name]; ok {
		if err := m.Select(instanceList, fmt.Sprintf(selectTemplate, pageSize, page*pageSize)); err != nil {
			return fmt.Errorf("[model] Error retrieving %s list from database: %s", m.name, err)
		}
	} else {
		return fmt.Errorf("[model] No list select statement template found for model %s", m.name)
	}
	return nil
}

// GetOne retrieves a model instance in base using its uuid
func (m *SQLModel) GetOne(instance interface{}, id uuid.UUID) error {
	if getOneStatement, ok := getOneStatements[m.name]; ok {
		if err := m.Get(instance, getOneStatement, id); err != nil {
			return fmt.Errorf("[model] Error retrieving %s %s from database: %s", m.name, id, err)
		}
	} else {
		return fmt.Errorf("[model] No get one statement found for model %s", m.name)
	}
	return nil
}

// Update changes a model instance in base using its uuid
func (m *SQLModel) Update(instance interface{}, id uuid.UUID) error {
	instanceMap := structs.Map(instance)
	instanceMap["prev_uuid"] = id
	if updateStatements, ok := updateStatements[m.name]; ok {
		if _, err := m.NamedExec(updateStatements, instanceMap); err != nil {
			return fmt.Errorf("[model] Error updating %s from database: %s", m.name, err)
		}
	} else {
		return fmt.Errorf("[model] No update statement found for model %s", m.name)
	}
	return nil
}

// CheckUUIDNotUsed checks if the UUID is alraedy used
func (m *SQLModel) CheckUUIDNotUsed(id uuid.UUID) error {
	rows, err := m.Queryx(fmt.Sprintf(`SELECT * FROM %s WHERE uuid='%s';`, m.name, id))
	if err != nil {
		return fmt.Errorf("[model] Error retrieving %s %s from database: %s", m.name, id, err)
	}
	if rows.Next() {
		return fmt.Errorf("[model] UUID %s already exist in table '%s'", id, m.name)
	}
	return nil
}

// GetModelName returns the model name
func (m *SQLModel) GetModelName() string {
	return m.name
}

// MockedModel is a mock of SQLModel for tests
type MockedModel struct {
	name string
}

// NewMockedModel creates a Model instance mock
func NewMockedModel(name string) (*MockedModel, error) {
	if _, ok := modelNames[name]; !ok {
		return nil, fmt.Errorf("Unknown model %s", name)
	}
	return &MockedModel{name}, nil
}

// Insert inserts a given model instance in base
func (m *MockedModel) Insert(instance interface{}) error {
	if _, ok := insertStatements[m.name]; ok {
	} else {
		return fmt.Errorf("[model] No insert statement found for model %s", m.name)
	}
	return nil
}

// List lists all model instances in base, pagination included
func (m *MockedModel) List(instanceList interface{}, page, pageSize int) error {
	if _, ok := selectTemplates[m.name]; ok {
	} else {
		return fmt.Errorf("[model] No list select statement template found for model %s", m.name)
	}
	return nil
}

// GetOne retrieves a model instance in base using its uuid
func (m *MockedModel) GetOne(instance interface{}, id uuid.UUID) error {
	if _, ok := getOneStatements[m.name]; ok {
	} else {
		return fmt.Errorf("[model] No get one statement found for model %s", m.name)
	}
	if id.String() == DevilMockUUID {
		return fmt.Errorf("[model] Runnin' With the Devil! sql: no rows in result set")
	}
	if id.String() == ProblemMockUUIDStr {
		ProblemMockUUID, _ := uuid.FromString(ProblemMockUUIDStr)

		// Fill struct Problem using reflection. This could be improved...
		u := reflect.New(reflect.TypeOf(ProblemMockUUID)).Elem().Interface()
		u = ProblemMockUUID

		i := reflect.ValueOf(instance).Elem()
		i.FieldByName("ID").Set(reflect.ValueOf(u))
		u = uuid.NewV4()
		i.FieldByName("Name").SetString("testName")
		i.FieldByName("Description").SetString("testDescription")

		return nil
	}
	return nil
}

// Update updates a model instance in base using its uuid
func (m *MockedModel) Update(instance interface{}, id uuid.UUID) error {
	return nil
}

// CheckUUIDNotUsed checks if the UUID is alraedy used
func (m *MockedModel) CheckUUIDNotUsed(id uuid.UUID) error {
	if id.String() == ProblemMockUUIDStr {
		return fmt.Errorf("[model] UUID %s already exist in table '%s'", id, m.name)
	}
	return nil
}

// GetModelName returns the model name
func (m *MockedModel) GetModelName() string {
	return m.name
}
