package clients

// HTTPMaxRetry indicates the number of
// retries to be carried out before giving up.
const HTTPMaxRetry = 9

// Client struct encapsulate the collection of
// external services
type Client struct {
	Resmgr   Resmgr
	Keystone Keystone
	Qbert    Qbert
	Executor Executor
	Segment  Segment
	HTTP     HTTP
}

// New creates the clients needed by the CLI
// to interact with the external services.
func New(fqdn string, proxy string) (Client, error) {

	http, err := NewHTTP(
		func(impl *HTTPImpl) { impl.Proxy = proxy },
		func(impl *HTTPImpl) { impl.Retry = HTTPMaxRetry })

	if err != nil {
		return Client{}, err
	}

	return Client{
		Resmgr:   NewResmgr(fqdn, http),
		Keystone: NewKeystone(fqdn, http),
		Qbert:    NewQbert(fqdn, http),
		Executor: ExecutorImpl{},
		Segment:  NewSegment(fqdn),
		HTTP:     http,
	}, nil
}
