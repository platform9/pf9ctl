package pmk

import (
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"go.uber.org/zap"
	"fmt"
)

// This files needs to be organized little better
func setupNode(hostOS string, exec cmdexec.Executor) (err error) {
	zap.S().Debug("Received a call to setup the node")

	if err := swapOff(exec); err != nil {
		return err
	}
	if err := swapOffFstab(exec, "/etc/fstab"); err != nil {
		return err
	}
	return nil
}

func swapOff(exec cmdexec.Executor) error {
	zap.S().Info("Disabling swap")

	_, err := exec.RunWithStdout("bash", "-c", "swapoff -a")
	return err
}

func swapOffFstab(exec cmdexec.Executor, file string) error {
	zap.S().Info("Removing swap in fstab")
	// match the 3rd column to have 'swap' and make sure the line isn't commented out already
	search := `^[[:space:]]*([^#][^[:space:]]+)[[:space:]]+([^[:space:]]+)[[:space:]]+(swap)[[:space:]](.*)$`
	replace := `#\1 \2 \3 \4`
	// also the expression is in the EXTENDED regexp (ERE) form not the BRE, so use the ERE form
	sedCmd := fmt.Sprintf("s/%s/%s/", search, replace)
	zap.S().Debug("Executing command ",sedCmd, file)
	stdout, err := exec.RunWithStdout("sed", "-E", "-i.bak", sedCmd, file)
	zap.S().Debug("Returned value: ", stdout)
	return err

}
