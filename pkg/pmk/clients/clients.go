package clients

const HTTPMaxRetry = 5

// Clients struct encapsulate the collection of
// external services
type Clients struct {
	Resmgr   Resmgr
	Keystone Keystone
	Qbert    Qbert
	Executor Executor
}

// New creates the clients needed by the CLI
// to interact with the external services.
func New(fqdn string) (Clients, error) {

	resmgr, _ := NewResmgr(fqdn)
	keystone, _ := NewKeystone(fqdn)
	qbert, _ := NewQbert(fqdn)

	return Clients{
		Resmgr:   resmgr,
		Keystone: keystone,
		Qbert:    qbert,
		Executor: ExecutorImpl{},
	}, nil
}
