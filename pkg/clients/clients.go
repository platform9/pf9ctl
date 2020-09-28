package clients

// Clients struct encapsulate the collection of
// external services
type Clients struct {
}

// NewClients creates the clients needed by the CLI
// to interact with the external services.
func NewClients() (Clients, error) {
	return Clients{}, nil
}
