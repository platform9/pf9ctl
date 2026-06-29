package keystone

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func assertValidJSON(t *testing.T, body string) {
	t.Helper()
	if !json.Valid([]byte(body)) {
		t.Fatalf("generated auth body is not valid JSON: %s", body)
	}
}

func TestBuildScopeSystem(t *testing.T) {
	scope := buildScope("service", true)
	assertValidJSON(t, "{"+scope+"}")
	if !strings.Contains(scope, `"system"`) || !strings.Contains(scope, `"all": true`) {
		t.Fatalf("expected a system scope, got: %s", scope)
	}
	if strings.Contains(scope, `"project"`) {
		t.Fatalf("system scope must not carry a project, got: %s", scope)
	}
}

func TestBuildScopeProjectByName(t *testing.T) {
	scope := buildScope("service", false)
	assertValidJSON(t, "{"+scope+"}")
	if !strings.Contains(scope, `"name": "service"`) {
		t.Fatalf("expected project-by-name scope, got: %s", scope)
	}
}

func TestBuildScopeProjectByID(t *testing.T) {
	id := "6d30c85c033247548d6d93b0056b266b"
	scope := buildScope(id, false)
	assertValidJSON(t, "{"+scope+"}")
	if !strings.Contains(scope, `"id": "`+id+`"`) {
		t.Fatalf("expected project-by-id scope, got: %s", scope)
	}
}

func TestBuildUserDomainDefault(t *testing.T) {
	dom := buildUserDomain("")
	assertValidJSON(t, "{"+dom+"}")
	if !strings.Contains(dom, `"id": "default"`) {
		t.Fatalf("empty domain must default to the default domain id, got: %s", dom)
	}
}

func TestBuildUserDomainByName(t *testing.T) {
	dom := buildUserDomain("domain-a")
	assertValidJSON(t, "{"+dom+"}")
	if !strings.Contains(dom, `"name": "domain-a"`) {
		t.Fatalf("expected domain-by-name, got: %s", dom)
	}
}

func TestBuildUserDomainByID(t *testing.T) {
	id := "2a73b8f597c04551a0fdc8e95544be8a"
	dom := buildUserDomain(id)
	assertValidJSON(t, "{"+dom+"}")
	if !strings.Contains(dom, `"id": "`+id+`"`) {
		t.Fatalf("expected domain-by-id, got: %s", dom)
	}
}

func TestBuildAuthBodyValidJSON(t *testing.T) {
	cases := []struct {
		name       string
		tenant     string
		mfa        string
		sys        bool
		userDomain string
	}{
		{"project-by-name", "service", "", false, ""},
		{"project-by-id", "6d30c85c033247548d6d93b0056b266b", "", false, ""},
		{"project-mfa", "service", "123456", false, ""},
		{"system", "service", "", true, ""},
		{"system-mfa", "service", "123456", true, ""},
		{"system-domain-name", "service", "", true, "domain-a"},
		{"system-domain-id", "service", "", true, "2a73b8f597c04551a0fdc8e95544be8a"},
		{"project-domain-name-mfa", "service", "123456", false, "domain-a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"auth": {%s, %s}}`,
				buildIdentity("someuser", "somepass", c.mfa, c.userDomain),
				buildScope(c.tenant, c.sys))
			assertValidJSON(t, body)
		})
	}
}
