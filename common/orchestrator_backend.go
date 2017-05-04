package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	uuid "github.com/satori/go.uuid"
)

// Orchestrator HTTP API routes
const (
	OrchestratorStatusUpdateRoute = "/update_status"
	OrchestratorLearnResultRoute  = "/learndone"
	OrchestratorPredResultRoute   = "/preddone"
)

// OrchestratorBackend describes Morpheo's orchestrator API
type OrchestratorBackend interface {
	UpdateUpletStatus(upletType, status string, upletID uuid.UUID) error
	PostLearnResult(learnupletID uuid.UUID, data io.Reader) error
}

// OrchestratorAPI is a wrapper around our orchestrator API
type OrchestratorAPI struct {
	OrchestratorBackend

	Hostname string
	Port     int
}

// UpdateUpletStatus changes the status field of a learnuplet/preduplet
func (o *OrchestratorAPI) UpdateUpletStatus(upletType string, status string, upletID uuid.UUID) error {
	if _, ok := ValidUplets[upletType]; !ok {
		return fmt.Errorf("[orchestrator-api] Uplet type \"%s\" is invalid. Allowed values are %s", upletType, ValidUplets)
	}
	if _, ok := ValidStatuses[status]; !ok {
		return fmt.Errorf("[orchestrator-api] Status \"%s\" is invalid. Allowed values are %s", status, ValidStatuses)
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
	url := fmt.Sprintf("http://%s:%d%s/%s/%s", o.Hostname, o.Port, route, upletID)

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
func (o *OrchestratorAPI) PostLearnResult(learnupletID uuid.UUID, data io.Reader) error {
	return o.postData(OrchestratorLearnResultRoute, learnupletID, data)
}
