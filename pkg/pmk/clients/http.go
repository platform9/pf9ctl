package clients

import (
	"net/http"
	"net/url"

	rhttp "github.com/hashicorp/go-retryablehttp"
)

type HTTP interface {
	Do(req *rhttp.Request) (*http.Response, error)
}

type HTTPImpl struct {
	Proxy       string
	Retry       int
	client      *rhttp.Client
	RetryPolicy rhttp.CheckRetry
	Backoff     rhttp.Backoff
	ProxyURL    *url.URL
}

func NewHTTP(options ...func(*HTTPImpl)) (*HTTPImpl, error) {
	resp := &HTTPImpl{}

	for _, option := range options {
		option(resp)
	}

	if resp.Proxy != "" {
		proxyURL, err := url.Parse(resp.Proxy)
		if err != nil {
			return nil, err
		}
		resp.ProxyURL = proxyURL
	}

	client := &rhttp.Client{}
	client.RetryMax = resp.Retry

	if resp.RetryPolicy != nil {
		client.CheckRetry = resp.RetryPolicy
	}

	if resp.Backoff != nil {
		client.Backoff = resp.Backoff
	}

	resp.client = client
	return resp, nil
}

// Do function simply calls the underlying client to make the request.
func (c HTTPImpl) Do(req *rhttp.Request) (*http.Response, error) {
	return c.client.Do(req)
}
