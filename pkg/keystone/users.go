// Copyright © 2021 The Platform9 Systems Inc.

package keystone

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Type definition for struct encapsulating user manager APIs.
type UserManagerAPI struct {
	Client  *http.Client
	BaseURL string
	Token   string
}

// Type definition for user information reported as part of the
// "get user" request.
type UserInfo struct {
	User struct {
		ID               string `json:"id"`
		DefaultProjectID string `json:"default_project_id"`
	} `json:"user"`
}

// Returns the user's default_project_id, or empty string if the user has none.
func GetUserDefaultProject(fqdn, token, userID string) (string, error) {
	url := fmt.Sprintf("%s/keystone/v3/users/%s", fqdn, userID)
	u_api := UserManagerAPI{newKeystoneHTTPClient(), url, token}
	return u_api.GetUserDefaultProject_API()
}

// User manager function to fetch the user's default project ID.
func (u_api *UserManagerAPI) GetUserDefaultProject_API() (string, error) {
	zap.S().Debug("Fetching default project for user")
	req, err := http.NewRequest("GET", u_api.BaseURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("X-Auth-Token", u_api.Token)

	resp, err := u_api.Client.Do(req)
	if err != nil {
		zap.S().Errorf("Failed to fetch user information, Error: %s", err)
		return "", fmt.Errorf("Failed to fetch user information, Error: %s", err)
	}
	defer resp.Body.Close()

	userInfo := UserInfo{}
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		zap.S().Errorf("Failed to decode user information, Error: %s", err)
		return "", fmt.Errorf("Failed to decode user information, Error: %s", err)
	}

	zap.S().Debug("default project ID: ", userInfo.User.DefaultProjectID)
	return userInfo.User.DefaultProjectID, nil
}
