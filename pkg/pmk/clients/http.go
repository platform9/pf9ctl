package clients

import (
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/rehttp"
)

type HTTP interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPImpl struct {
	Proxy    string
	Retry    int
	client   *http.Client
	ProxyURL *url.URL
}

func NewHTTP(options ...func(*HTTPImpl)) (*HTTPImpl, error) {
	resp := &HTTPImpl{}

	for _, option := range options {
		option(resp)
	}

	var transport http.RoundTripper
	if resp.Proxy != "" {
		proxyURL, err := url.Parse(resp.Proxy)
		if err != nil {
			return nil, err
		}

		resp.ProxyURL = proxyURL
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	} else {
		transport = http.DefaultTransport
	}

	t := rehttp.NewTransport(transport, rehttp.RetryAll(
		rehttp.RetryAny(
			rehttp.RetryTemporaryErr(),
			rehttp.RetryStatuses(400, 404),
		),
		rehttp.RetryMaxRetries(resp.Retry),
	),
		// rehttp.ExpJitterDelay(time.Second*time.Duration(2), time.Second*time.Duration(60))
		rehttp.ConstDelay(time.Second*time.Duration(10)))

	resp.client = &http.Client{Transport: t}
	return resp, nil
}

// Do function simply calls the underlying client to make the request.
func (c HTTPImpl) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}
