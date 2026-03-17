// Copyright © 2020 The Platform9 Systems Inc.

package keystone

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func keystoneRootCAs() *x509.CertPool {
	rootCAs := x509.NewCertPool()

	// Prefer the DU cert if present (installed by pcdctl).
	for _, p := range []string{
		"/etc/pki/ca-trust/source/anchors/du.crt",
		"/usr/local/share/ca-certificates/du.crt",
	} {
		if b, err := os.ReadFile(p); err == nil {
			_ = rootCAs.AppendCertsFromPEM(b)
		}
	}

	// Load OS CA bundle directly (avoids per-process caching).
	for _, p := range []string{
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // RHEL-family
		"/etc/ssl/certs/ca-certificates.crt",               // Debian-family
	} {
		if b, err := os.ReadFile(p); err == nil && rootCAs.AppendCertsFromPEM(b) {
			return rootCAs
		}
	}

	// Fallback: best-effort system pool (may be cached).
	if sysPool, sysErr := x509.SystemCertPool(); sysErr == nil && sysPool != nil {
		return sysPool
	}

	return rootCAs
}

func newKeystoneHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			RootCAs:    keystoneRootCAs(),
			MinVersion: tls.VersionTLS12,
		},
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

type KeystoneAuth struct {
	DUFqdn    string
	Token     string
	UserID    string
	ProjectID string
	Email     string
}

type Keystone interface {
	GetAuth(username, password, tenant string, mfa string) (KeystoneAuth, error)
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
	tenant string,
	mfa string) (auth KeystoneAuth, err error) {

	zap.S().Debugf("Received a call to fetch keystone authentication for fqdn: %s and user: %s and tenant: %s, mfa_token: %s\n", k.fqdn, username, tenant, mfa)

	url := fmt.Sprintf("%s/keystone/v3/auth/tokens?nocatalog", k.fqdn)

	var body string

	//parsing the id or name passed in the tenant[] field
	_, matcherr := uuid.Parse(tenant)

	if mfa != "" {
		// if err is nil, then it may be the correct ID
		if matcherr == nil {
			body = fmt.Sprintf(`{
                	"auth": {
                        	"identity": {
                                	"methods": ["password", "totp"],
                                	"password": {
                                        	"user": {
                                                	"name": "%s",
                                                	"domain": {"id": "default"},
                                                	"password": "%s"
                                        	}
                                	},
					"totp": {
						"user": {
                                  			"name": "%s",
                                  			"domain": {
                                          			"id": "default"
                                  			},
                          				"passcode": "%s"
                          			}
					}
                        	},
                        	"scope": {
                                	"project": {
                                        	"id": "%s",
                                        	"domain": {"id": "default"}
                                	}
                        	}
                  	}
        	}`, username, password, username, mfa, tenant)
		} else { // if there is an error, then name is passed and we are accepting name field then
			body = fmt.Sprintf(`{
				"auth": {
						"identity": {
								"methods": ["password", "totp"],
								"password": {
										"user": {
												"name": "%s",
												"domain": {"id": "default"},
												"password": "%s"
										}
								},
				"totp": {
					"user": {
										  "name": "%s",
										  "domain": {
												  "id": "default"
										  },
									  "passcode": "%s"
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
			}`, username, password, username, mfa, tenant)
		}
	} else {
		if matcherr == nil {
			body = fmt.Sprintf(`{
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
							"id": "%s",
							"domain": {"id": "default"}
						}
					}
				}
			}`, username, password, tenant)
		} else {
			body = fmt.Sprintf(`{
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
		}
	}

	client := newKeystoneHTTPClient()

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return auth, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		zap.S().Debugf("Error calling keystone API:%s\n", err.Error())
		return auth, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		zap.S().Debugf("Error in StatusCode:%d\n", resp.StatusCode)
		return auth, fmt.Errorf("Unable to get keystone token, status: %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		zap.S().Debugf("Error in decoding payload\n")
		return auth, fmt.Errorf("Unable to decode the payload")
	}
	t := payload["token"].(map[string]interface{})
	project := t["project"].(map[string]interface{})
	user := t["user"].(map[string]interface{})
	token := resp.Header["X-Subject-Token"][0]

	zap.S().Debugf("returning successfully\n")

	return KeystoneAuth{
		DUFqdn:    k.fqdn,
		Token:     token,
		UserID:    user["id"].(string),
		ProjectID: project["id"].(string),
		Email:     user["name"].(string),
	}, nil
}
