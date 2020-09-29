package pmk

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/platform9/pf9ctl/pkg/log"
)

// KeystoneAuth represents user authenticated information.
type KeystoneAuth struct {
	Token     string
	ProjectID string
	UserID    string
}

// GetKeystoneAuth returns the keystone credentials for the
// host.
func GetKeystoneAuth(host, username, password, tenant string) (KeystoneAuth, error) {
	log.Info.Printf("Received a call to get keystone authentication for host: %s\n", host)

	return getKeystoneAuth(host, username, password, tenant)
}

func getKeystoneAuth(host, username, password, tenant string) (KeystoneAuth, error) {
	auth := KeystoneAuth{}
	url := fmt.Sprintf("%s/keystone/v3/auth/tokens?nocatalog", host)

	// Decoding base64 encoded password
	decodedBytePassword, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return auth, err
	}
	decodedPassword := string(decodedBytePassword)

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
	}`, username, decodedPassword, tenant)

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

	auth = KeystoneAuth{
		Token:     token,
		UserID:    user["id"].(string),
		ProjectID: project["id"].(string),
	}

	return auth, nil
}
