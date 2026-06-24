package keystone_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	. "github.com/platform9/pf9ctl/pkg/keystone"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
)

var projectsInfo string = `{"projects": [{"id": "8f1d2c3b4a5e6f7081920a1b2c3d4e5f", "name": "service"}]}`

func TestGetProjectID(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		Equals(t, req.URL.String(), "http://example.com?name=service")
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(projectsInfo)),
			Header:     make(http.Header),
		}
	})

	p_api := ProjectManagerAPI{client, "http://example.com", "token"}
	id, err := p_api.GetProjectID_API("service")
	Ok(t, err)
	Equals(t, "8f1d2c3b4a5e6f7081920a1b2c3d4e5f", id)
}

// A value that already looks like an id resolves without an API call.
func TestGetProjectIDUUIDShortCircuit(t *testing.T) {
	id, err := GetProjectID("https://example.platform9.horse", "token", "8f1d2c3b4a5e6f7081920a1b2c3d4e5f")
	Ok(t, err)
	Equals(t, "8f1d2c3b4a5e6f7081920a1b2c3d4e5f", id)
}

var userInfoWithDefaultProject string = `{"user": {"id": "u1", "default_project_id": "proj-default-id"}}`
var userInfoNoDefaultProject string = `{"user": {"id": "u1"}}`

func TestGetUserDefaultProject(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(userInfoWithDefaultProject)),
			Header:     make(http.Header),
		}
	})

	u_api := UserManagerAPI{client, "http://example.com", "token"}
	id, err := u_api.GetUserDefaultProject_API()
	Ok(t, err)
	Equals(t, "proj-default-id", id)
}

func TestGetUserDefaultProjectEmpty(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(userInfoNoDefaultProject)),
			Header:     make(http.Header),
		}
	})

	u_api := UserManagerAPI{client, "http://example.com", "token"}
	id, err := u_api.GetUserDefaultProject_API()
	Ok(t, err)
	Equals(t, "", id)
}
