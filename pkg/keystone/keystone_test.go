package keystone_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
	. "github.com/platform9/pf9ctl/pkg/keystone"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
       )

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

var serviceInfo string = `{"services": [{"description": "Links to region specific resources hosted on a DU","name": "regioninfo", "id": "6d30c85c033247548d6d93b0056b266b", "type": "regioninfo", "enabled": true, "links": { "self": "https://example.platform9.horse/keystone/v3/services/6d30c85c033247548d6d93b0056b266b"}}], "links": { "next": null, "self": "https://example.platform9.horse/keystone/v3/services?type=regionInfo", "previous": null}}`

var serviceID_expected string = "6d30c85c033247548d6d93b0056b266b"

// Tests the API to fetch cluster FQDN.
func TestGetServiceID(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test request parameters
		Equals(t, req.URL.String(), "http://example.com?type=regionInfo")
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString(serviceInfo)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	s_api := ServiceManagerAPI{client, "http://example.com", "token"}
	serviceID_actual, err := s_api.GetServiceID_API("regionInfo")
	Ok(t, err)
	Equals(t, serviceID_expected, serviceID_actual)
}

var endpointInfo string = `{
  "endpoints": [
    {
      "id": "08732d8b2c29499883d96b4e63a6abd0",
      "interface": "public",
      "region_id": "region1",
      "service_id": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example-region1.platform9.horse/links/",
      "enabled": true,
      "region": "region1",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/08732d8b2c29499883d96b4e63a6abd0"
      }
    },
    {
      "id": "36b1a102daa84360b7dc55c09b85b6fd",
      "interface": "internal",
      "region_id": "region1",
      "service_id": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example-region1.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region1",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/36b1a102daa84360b7dc55c09b85b6fd"
      }
    },
    {
      "id": "3d8cbf02660648d6bfd3b18697372488",
      "interface": "admin",
      "region_id": "region2",
      "service_id": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region2",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/3d8cbf02660648d6bfd3b18697372488"
      }
    },
    {
      "id": "4310d6bd9b34451b8d04ec1992ecbe70",
      "interface": "internal",
      "region_id": "region2",
      "service_id": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region2",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/4310d6bd9b34451b8d04ec1992ecbe70"
      }
    },
    {
      "id": "d55e08a83f2141c092d2d0e339bf501e",
      "interface": "public",
      "region_id": "region2",
      "service_id": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example.platform9.horse/links/",
      "enabled": true,
      "region": "region2",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/d55e08a83f2141c092d2d0e339bf501e"
      }
    },
    {
      "id": "ff1634c5e83042f49f6cca837c254aa3",
      "interface": "admin",
      "region_id": "region1",
      "service_id": "6d30c85c033247548d6d93b0056b266b",
      "url": "https://example-region1.platform9.horse/private/links.json",
      "enabled": true,
      "region": "region1",
      "links": {
        "self": "https://example-region1.platform9.horse/keystone/v3/endpoints/ff1634c5e83042f49f6cca837c254aa3"
      }
    }
  ],
  "links": {
    "next": null,
    "self": "https://example-region1.platform9.horse/keystone/v3/endpoints?service_id=6d30c85c033247548d6d93b0056b266b",
    "previous": null
  }
}`

var region1_endpoint_expected string = "example-region1.platform9.horse"
var region2_endpoint_expected string = "example.platform9.horse"

// Tests the API to fetch cluster FQDN.
func TestGetEndpointForRegion(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test request parameters
		Equals(t, req.URL.String(), "http://example.com?service_id=6d30c85c033247548d6d93b0056b266b")
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString(endpointInfo)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	e_api := EndpointManagerAPI{client, "http://example.com", "token"}

	// Test for region1
	endpoint_actual, err := e_api.GetEndpointForRegion_API("region1", "6d30c85c033247548d6d93b0056b266b")
	Ok(t, err)
	Equals(t, region1_endpoint_expected, endpoint_actual)


	// Test for region2
	endpoint_actual, err = e_api.GetEndpointForRegion_API("region2", "6d30c85c033247548d6d93b0056b266b")
	Ok(t, err)
	Equals(t, region2_endpoint_expected, endpoint_actual)
}
