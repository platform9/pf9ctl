// Copyright © 2021 The Platform9 Systems Inc.

package keystone

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Type definition for struct encapsulating project manager APIs.
type ProjectManagerAPI struct {
	Client  *http.Client
	BaseURL string
	Token   string
}

// Type definition for project information reported as part of the
// "get projects" request.
type ProjectsInfo struct {
	Projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"projects"`
}

// Resolves a project name to its ID; a value that is already a UUID is returned unchanged.
func GetProjectID(fqdn, token, project string) (string, error) {
	if _, err := uuid.Parse(project); err == nil {
		return project, nil
	}

	url := fmt.Sprintf("%s/keystone/v3/projects", fqdn)
	p_api := ProjectManagerAPI{newKeystoneHTTPClient(), url, token}
	return p_api.GetProjectID_API(project)
}

// Project manager function to fetch the project ID for a given name.
func (p_api *ProjectManagerAPI) GetProjectID_API(name string) (string, error) {
	zap.S().Debug("Fetching project ID for ", name)
	req, err := http.NewRequest("GET", p_api.BaseURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("X-Auth-Token", p_api.Token)

	q := req.URL.Query()
	q.Add("name", name)
	req.URL.RawQuery = q.Encode()

	resp, err := p_api.Client.Do(req)
	if err != nil {
		zap.S().Errorf("Failed to fetch project information for %s, Error: %s", name, err)
		return "", fmt.Errorf("Failed to fetch project information for %s, Error: %s", name, err)
	}
	defer resp.Body.Close()

	projectsInfo := ProjectsInfo{}
	err = json.NewDecoder(resp.Body).Decode(&projectsInfo)
	if err != nil {
		zap.S().Errorf("Failed to decode project information, Error: %s", err)
		return "", fmt.Errorf("Failed to decode project information, Error: %s", err)
	}

	if len(projectsInfo.Projects) == 0 {
		return "", fmt.Errorf("no project found with name %s", name)
	}

	projectID := projectsInfo.Projects[0].ID
	zap.S().Debug("project ID: ", projectID)
	return projectID, nil
}
