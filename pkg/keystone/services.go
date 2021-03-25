// Copyright © 221 The Platform9 Systems Inc.

package keystone

import (
	"encoding/json"
	"fmt"
	"net/http"
	"go.uber.org/zap"
)

// Type definition for struct encapsulating service manager APIs.
type ServiceManagerAPI struct {
	Client  *http.Client
	BaseURL string
	Token   string
}

// Type definition for services information that is reported as
// part of the "get services" request.
type ServicesInfo struct {
	Services []struct {
		Description string `json:"description"`
		Links       struct {
			Self string `json:"self"`
		} `json:"links"`
		Enabled bool   `json:"enabled"`
		Type    string `json:"type"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	} `json:"services"`
	Links struct {
		Self     string      `json:"self"`
		Previous interface{} `json:"previous"`
		Next     interface{} `json:"next"`
	} `json:"links"`
}

// Fetches the ID for service registered in the keystone database.
func GetServiceID(
	fqdn string, //DU fqdn
	auth KeystoneAuth, //Auth info
	name string, //Service name
) (string, error) {

	zap.S().Debug("Fetching service ID for service: ", name)

	// Form the URL
	url := fmt.Sprintf("%s/keystone/v3/services", fqdn)

	// Generate the http client object
	client := &http.Client{}

	// Create the context to invoke the service manager API.
	s_api := ServiceManagerAPI{client, url, auth.Token}

	// Invoke the actual "get services" API.
	ID, err := s_api.GetServiceID_API(name)
	if err != nil {
		return "", err
	}

	zap.S().Debug("service ID : ", ID)
	return ID, nil
}

// Service manager function to fetch service ID
func (s_api *ServiceManagerAPI) GetServiceID_API(
	name string,
) (string, error) {

	zap.S().Debug("Fetching service ID for ", name)
	req, err := http.NewRequest("GET", s_api.BaseURL, nil)

	// Add keystone token in the header.
	req.Header.Add("X-Auth-Token", s_api.Token)

	// Add the query parameter "type"
	q := req.URL.Query()
	q.Add("type", name)
	req.URL.RawQuery = q.Encode()

	resp, err := s_api.Client.Do(req)
	if err != nil {
		zap.S().Errorf("Failed to fetch service information for service %s, Error: %s", name, err)
		return "", fmt.Errorf("Failed to fetch service information for service %s, Error: %s",name, err)
	}
	defer resp.Body.Close()

	serviceInfo := ServicesInfo{}
	// Response is received as slice of services.
	err = json.NewDecoder(resp.Body).Decode(&serviceInfo)
	if err != nil {
		zap.S().Errorf("Failed to decode service information, Error: %s", err)
		return "", fmt.Errorf("Failed to decode service information, Error: %s", err)
	}

	// There is supposed to be only one service per name.
	// Pick the ID from the first instance in the slice.
	service_id := serviceInfo.Services[0].ID
	zap.S().Debug("service ID : .", service_id)
	return service_id, nil
}
