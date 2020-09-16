package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type KeystoneAuthRequest struct {
	Auth *KeystoneAuth `json:"auth"`
}

type KeystoneAuth struct {
	Identity *KeystoneIdentity `json:"identity"`
}

type KeystoneIdentity struct {
	Password *KeystoneIdentityPassword `json:"password"`
	Methods  []string                  `json:"methods"`
}

type KeystoneIdentityPassword struct {
	User *KeystoneIdentityPasswordUser `json:"user"`
}

type KeystoneIdentityPasswordUser struct {
	Domain   map[string]string `json:"domain"`
	Password string            `json:"password"`
	Name     string            `json:"name"`
}

type KeystoneAuthResponse struct {
	Token KeystoneToken `json:"token"`
}

type KeystoneToken struct {
	// Note: this is just a partial implementation
	ExpiresAt time.Time `json:"expires_at"`
}

type KeystoneClient struct {

	// endpoint should contain the base path of keystone.
	//
	// Example: https://some-du.platform9.horse/keystone/v3
	endpoint string

	httpClient *http.Client
}

func NewKeystoneClient(endpoint string, httpClient *http.Client) *KeystoneClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &KeystoneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		httpClient: httpClient,
	}
}

func (k *KeystoneClient) Auth(credentials Credentials) (TokenInfo, error) {
	keystoneReq := credentialsToKeystoneAuthRequest(credentials)

	reqBody, err := json.Marshal(keystoneReq)
	if err != nil {
		return TokenInfo{}, err
	}

	// TODO(erwin) use certificate data from kubeconfig (might need upstream work)
	// TODO(erwin) give a better error for invalid login error: failed to authenticate: received a 401 from keystone: {"error": {"message": "The request you have made requires authentication.", "code": 401, "title": "Unauthorized"}}
	resp, err := k.httpClient.Post(k.endpoint+"/auth/tokens", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return TokenInfo{}, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return TokenInfo{}, err
	}

	if resp.StatusCode >= 400 {
		return TokenInfo{}, fmt.Errorf("failed to authenticate: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	tokenResp := &KeystoneAuthResponse{}
	err = json.Unmarshal(respBody, tokenResp)
	if err != nil {
		return TokenInfo{}, err
	}

	token := resp.Header.Get("x-subject-token")
	if token == "" {
		return TokenInfo{}, errors.New("keystone authentication response did not contain the 'x-subject-token' header")
	}

	return TokenInfo{
		Token:     token,
		ExpiresAt: tokenResp.Token.ExpiresAt,
	}, nil
}

func credentialsToKeystoneAuthRequest(credentials Credentials) *KeystoneAuthRequest {
	return &KeystoneAuthRequest{
		Auth: &KeystoneAuth{
			Identity: &KeystoneIdentity{
				Methods: []string{"password"},
				Password: &KeystoneIdentityPassword{
					User: &KeystoneIdentityPasswordUser{
						Domain: map[string]string{
							"id": "default",
						},
						Password: credentials.Password,
						Name:     credentials.Username,
					},
				},
			},
		},
	}
}
