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
	"log"
	"net/http"

	"github.com/MorpheoOrg/go-morpheo/common"
	uuid "github.com/satori/go.uuid"
)

// Orchestrator HTTP API routes
const (
	OrchestratorStatusUpdateRoute = "/update_status"
	OrchestratorLearnResultRoute  = "/learndone"
	OrchestratorPredResultRoute   = "/preddone"
)

// Perfuplet describes the response of compute to the orchestrator
type Perfuplet struct {
	Status    string             `json:"status"`
	Perf      float64            `json:"perf"`
	TrainPerf map[string]float64 `json:"train_perf"`
	TestPerf  map[string]float64 `json:"test_perf"`
}

// Orchestrator describes Morpheo's orchestrator API
type Orchestrator interface {
	UpdateUpletStatus(upletType, status string, upletID uuid.UUID) error
	PostLearnResult(learnupletID uuid.UUID, perfuplet Perfuplet) error
}

// OrchestratorAPI is a wrapper around our orchestrator API
type OrchestratorAPI struct {
	Orchestrator

	Hostname string
	Port     int
}

// UpdateUpletStatus changes the status field of a learnuplet/preduplet
func (o *OrchestratorAPI) UpdateUpletStatus(upletType string, status string, upletID uuid.UUID) error {
	if _, ok := common.ValidUplets[upletType]; !ok {
		return fmt.Errorf("[orchestrator-api] Uplet type \"%s\" is invalid. Allowed values are %s", upletType, common.ValidUplets)
	}
	if _, ok := common.ValidStatuses[status]; !ok {
		return fmt.Errorf("[orchestrator-api] Status \"%s\" is invalid. Allowed values are %s", status, common.ValidStatuses)
	}
	url := fmt.Sprintf("http://%s:%d%s/%s/%s", o.Hostname, o.Port, OrchestratorStatusUpdateRoute, upletType, upletID)

	payload, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return fmt.Errorf("[orchestrator-api] Error JSON-marshaling status update payload: %s", url, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("[orchestrator-api] Error building status update POST request against %s: %s", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("[orchestrator-api] Error performing status update POST request against %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[orchestrator-api] Unexpected status code (%s): status update POST request against %s", resp.Status, url)
	}
	return nil
}

func (o *OrchestratorAPI) postData(route string, upletID uuid.UUID, data io.Reader) error {
	url := fmt.Sprintf("http://%s:%d%s/%s", o.Hostname, o.Port, route, upletID)

	req, err := http.NewRequest(http.MethodPost, url, data)
	if err != nil {
		return fmt.Errorf("[orchestrator-api] Error building result POST request against %s: %s", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("[orchestrator-api] Error performing result POST request against %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[orchestrator-api] Unexpected status code (%s): result POST request against %s", resp.Status, url)
	}
	return nil
}

// PostLearnResult forwards a JSON-formatted learn result to the orchestrator HTTP API
func (o *OrchestratorAPI) PostLearnResult(learnupletID uuid.UUID, perfuplet Perfuplet) error {
	dataBytes, err := json.Marshal(perfuplet)
	if err != nil {
		return fmt.Errorf("Error marshaling perfuplet to JSON: %s", perfuplet)
	}
	data := bytes.NewReader(dataBytes)
	return o.postData(OrchestratorLearnResultRoute, learnupletID, data)
}

// OrchestratorAPIMock mocks the Orchestrator API, always returning ok to update queries except for
// given "unexisting" pred/learn uplet with a given UUID
type OrchestratorAPIMock struct {
	Orchestrator

	UnexistingUplet string
}

// NewOrchestratorAPIMock returns with a mock of the Orchestrator API
func NewOrchestratorAPIMock() (s *OrchestratorAPIMock) {
	return &OrchestratorAPIMock{
		UnexistingUplet: "ea408171-0205-475e-8962-a02855767260",
	}
}

// UpdateUpletStatus returns nil except if OrchestratorAPIMock.UnexistingUpletID is passed
func (o *OrchestratorAPIMock) UpdateUpletStatus(upletType, status string, upletID uuid.UUID) error {
	if upletID.String() != o.UnexistingUplet {
		log.Printf("[orchestrator-mock] Received update status for %s-uplet %s. Status: %s", upletType, upletID, status)
		return nil
	}
	return fmt.Errorf("[orchestrator-mock][status-update] Unexisting uplet %s", upletID)
}

// PostLearnResult returns nil except if OrchestratorAPIMock.UnexistingUpletID is passed
func (o *OrchestratorAPIMock) PostLearnResult(learnupletID uuid.UUID, perfuplet Perfuplet) error {
	if learnupletID.String() != o.UnexistingUplet {
		log.Printf("[orchestrator-mock] Received learn result for learn-uplet %s: \n %s", learnupletID, perfuplet)
		return nil
	}
	return fmt.Errorf("[orchestrator-mock][status-update] Unexisting uplet %s", learnupletID)
}
