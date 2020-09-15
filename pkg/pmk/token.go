package pmk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// KeystoneAuth represents user authenticated information.
type KeystoneAuth struct {
	Token     string
	ProjectID string
	UserID    string
}

func GetKeystoneAuth(host, username, password, tenant string) (KeystoneAuth, error) {
	return getKeystoneAuth(host, username, password, tenant)
}

func getKeystoneAuth(host, username, password, tenant string) (KeystoneAuth, error) {
	url := fmt.Sprintf("%s/keystone/v3/auth/tokens?nocatalog", host)

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
		return KeystoneAuth{}, err
	}

	if resp.StatusCode != 201 {
		return KeystoneAuth{}, fmt.Errorf("Unable to get keystone token, status: %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&payload)
	if err != nil {
		return KeystoneAuth{}, fmt.Errorf("Unable to decode the payload")
	}

	t := payload["token"].(map[string]interface{})
	project := t["project"].(map[string]interface{})
	user := t["user"].(map[string]interface{})

	token := resp.Header["X-Subject-Token"][0]
	return KeystoneAuth{
		Token:     token,
		UserID:    user["id"].(string),
		ProjectID: project["id"].(string),
	}, nil
}
