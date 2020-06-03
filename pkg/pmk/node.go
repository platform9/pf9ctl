package pmk

import (
	"net"
)

// PrepNode sets up prerequisites for k8s stack
func PrepNode(node net.IPAddr) error {

	return nil
}

// AttachNode attaches a prepped node to a specified cluster
func AttachNode(node net.IPAddr, cluster *Cluster) error {

	return nil
}
