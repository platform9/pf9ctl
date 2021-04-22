// Copyright © 2020 The Platform9 Systems Inc.

package keystone

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type KeystoneAuth struct {
	Token     string
	UserID    string
	ProjectID string
	Email     string
}

type Keystone interface {
	GetAuth(username, password, tenant string) (KeystoneAuth, error)
}

type KeystoneImpl struct {
	fqdn string
}

func NewKeystone(fqdn string) Keystone {
	return KeystoneImpl{fqdn}
}

func (k KeystoneImpl) GetAuth(
	username,
	password,
	tenant string) (auth KeystoneAuth, err error) {

	zap.S().Debugf("Received a call to fetch keystone authentication for fqdn: %s and user: %s and tenant: %s\n", k.fqdn, username, tenant)

	url := fmt.Sprintf("%s/keystone/v3/auth/tokens?nocatalog", k.fqdn)

	body := fmt.Sprintf(`{
		"auth": {
			"identity": {
				"methods": ["password"],
				"password": {
					"user": {
						"name": "%s",
						"domain": {"id": "default"},
						"password": "%s"
					}
				}
			},
			"scope": {
				"project": {
					"name": "%s",
					"domain": {"id": "default"}
				}
			}
		}
	}`, username, password, tenant)

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return auth, err
	}

	if resp.StatusCode != 201 {
		return auth, fmt.Errorf("Unable to get keystone token, status: %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		return auth, fmt.Errorf("Unable to decode the payload")
	}
	t := payload["token"].(map[string]interface{})
	project := t["project"].(map[string]interface{})
	user := t["user"].(map[string]interface{})
	token := resp.Header["X-Subject-Token"][0]

	return KeystoneAuth{
		Token:     token,
		UserID:    user["id"].(string),
		ProjectID: project["id"].(string),
		Email:     user["name"].(string),
	}, nil
}
