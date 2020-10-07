package clients

const HTTPMaxRetry = 5

// Clients struct encapsulate the collection of
// external services
type Client struct {
	Resmgr   Resmgr
	Keystone Keystone
	Qbert    Qbert
	Executor Executor
	Segment  Segment
}

// New creates the clients needed by the CLI
// to interact with the external services.
func New(fqdn string) (Client, error) {
	return Client{
		Resmgr:   NewResmgr(fqdn),
		Keystone: NewKeystone(fqdn),
		Qbert:    NewQbert(fqdn),
		Executor: ExecutorImpl{},
		Segment:  NewSegment(fqdn),
	}, nil
}
