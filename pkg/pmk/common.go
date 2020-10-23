package pmk

import (
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk/clients"
)
// This files needs to be organized little better
func setupNode(hostOS string, exec clients.Executor) (err error) {
	log.Debug("Received a call to setup the node")

	if err := swapOff(exec); err != nil {
		return err
	}
	return nil
}


func swapOff(exec clients.Executor) error {
	log.Info("Disabling swap")

	_, err := exec.RunWithStdout("bash", "-c", "swapoff -a")
	return err
}
