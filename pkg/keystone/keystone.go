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
		"/etc/ssl/certs/ca-certificates.crt",                // Debian-family
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
	GetAuth(username, password, tenant string, mfa string, systemScope bool, userDomain string) (KeystoneAuth, error)
}

// Returns the keystone identity-domain block (by id if UUID, else by name).
// Empty defaults to the "default" domain by id, since its name is "Default".
func buildUserDomain(userDomain string) string {
	if userDomain == "" {
		return `"domain": {"id": "default"}`
	}
	if _, err := uuid.Parse(userDomain); err == nil {
		return fmt.Sprintf(`"domain": {"id": "%s"}`, userDomain)
	}
	return fmt.Sprintf(`"domain": {"name": "%s"}`, userDomain)
}

// Returns the keystone "identity" block; password+totp methods when mfa is set, else password only.
func buildIdentity(username, password, mfa, userDomain string) string {
	dom := buildUserDomain(userDomain)
	if mfa != "" {
		return fmt.Sprintf(`"identity": {
			"methods": ["password", "totp"],
			"password": {"user": {"name": "%s", %s, "password": "%s"}},
			"totp": {"user": {"name": "%s", %s, "passcode": "%s"}}
		}`, username, dom, password, username, dom, mfa)
	}
	return fmt.Sprintf(`"identity": {
		"methods": ["password"],
		"password": {"user": {"name": "%s", %s, "password": "%s"}}
	}`, username, dom, password)
}

// Returns the keystone "scope" block; system-scoped when systemScope is set, else project-scoped to tenant.
func buildScope(tenant string, systemScope bool) string {
	if systemScope {
		return `"scope": {"system": {"all": true}}`
	}
	if _, err := uuid.Parse(tenant); err == nil {
		return fmt.Sprintf(`"scope": {"project": {"id": "%s", "domain": {"id": "default"}}}`, tenant)
	}
	return fmt.Sprintf(`"scope": {"project": {"name": "%s", "domain": {"id": "default"}}}`, tenant)
}

// Picks the project id a system-scoped actor operates on (its token carries none).
// Precedence: explicit tenant, then user default project, then the "service" project.
func resolveSystemTargetProject(fqdn, token, tenant, userID string) (string, error) {
	if tenant != "" {
		id, err := GetProjectID(fqdn, token, tenant)
		if err != nil {
			return "", fmt.Errorf("Unable to resolve target project %s, Error: %s", tenant, err)
		}
		return id, nil
	}

	if userID != "" {
		id, err := GetUserDefaultProject(fqdn, token, userID)
		if err != nil {
			zap.S().Debugf("Unable to fetch user default project, falling back to service: %s", err)
		} else if id != "" {
			zap.S().Debugf("Using user default project %s as target", id)
			return id, nil
		}
	}

	id, err := GetProjectID(fqdn, token, "service")
	if err != nil {
		return "", fmt.Errorf("Unable to resolve a target project (no tenant, no user default project, service lookup failed), Error: %s", err)
	}
	return id, nil
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
	mfa string,
	systemScope bool,
	userDomain string) (auth KeystoneAuth, err error) {

	zap.S().Debugf("Received a call to fetch keystone authentication for fqdn: %s and user: %s and tenant: %s, mfa_token: %s, system_scope: %t, user_domain: %s\n", k.fqdn, username, tenant, mfa, systemScope, userDomain)

	url := fmt.Sprintf("%s/keystone/v3/auth/tokens?nocatalog", k.fqdn)

	body := fmt.Sprintf(`{"auth": {%s, %s}}`,
		buildIdentity(username, password, mfa, userDomain),
		buildScope(tenant, systemScope))

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
	t, _ := payload["token"].(map[string]interface{})

	var token string
	if subjectToken := resp.Header["X-Subject-Token"]; len(subjectToken) > 0 {
		token = subjectToken[0]
	}

	// A system-scoped token has no "project" block; leave ProjectID empty.
	var projectID string
	if project, ok := t["project"].(map[string]interface{}); ok {
		projectID, _ = project["id"].(string)
	}

	var userID, email string
	if user, ok := t["user"].(map[string]interface{}); ok {
		userID, _ = user["id"].(string)
		email, _ = user["name"].(string)
	}

	if systemScope {
		projectID, err = resolveSystemTargetProject(k.fqdn, token, tenant, userID)
		if err != nil {
			return auth, err
		}
	}

	zap.S().Debugf("returning successfully\n")

	return KeystoneAuth{
		DUFqdn:    k.fqdn,
		Token:     token,
		UserID:    userID,
		ProjectID: projectID,
		Email:     email,
	}, nil
}
