package pmk

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/pmk/clients"
)

// Host interface exposes the functions
// required to setup the host correctly.
type Host interface {
	Setup() error
	InstallPackage(...string) error
	EnableNTP() error
	SwapOff() error
	String() string
	PackagePresent(name string) bool
}

//Redhat host encapsulates menthods
//required to bootup the redhat host.
type Redhat struct {
	exec clients.Executor
}

func (h Redhat) String() string {
	return "redhat"
}

// PackagePresent checks for the package in the Redhat host.
func (h Redhat) PackagePresent(name string) bool {
	err := h.exec.Run("bash", "-c", fmt.Sprintf("yum list | grep -i '%s'", name))
	return err == nil
}

// Setup installes the necessary packages for the
// host to start functioning correctly.
func (h Redhat) Setup() error {
	return h.InstallPackage("libselinux-python")
}

// InstallPackage installs packages provided in the arguments
// for the Redhat host.
func (h Redhat) InstallPackage(names ...string) error {
	packages := ""
	for _, name := range names {
		packages += " " + name
	}
	return h.exec.Run(
		"bash",
		"-c",
		fmt.Sprintf("yum install -y %s", packages))
}

//EnableNTP enables the NTP service for the Redhat host.
func (h Redhat) EnableNTP() error {
	err := h.InstallPackage("ntpd")
	err = h.exec.Run("bash", "-c", "systemctl enable --now ntp")
	return err
}

//SwapOff disables the swap for the Redhat host.
func (h Redhat) SwapOff() error {
	return h.exec.Run("bash", "-c", "swapoff -a")
}

type Debian struct {
	exec clients.Executor
}

//Setup sets the host up with required packages for
//the preping the node up.
func (h Debian) Setup() error {
	err := h.exec.Run("bash", "-c", "apt-get update")
	err = h.InstallPackage("curl", "uuid-runtime", "software-properties-common", "logrotate")
	return err
}

// InstallPackage installs packages provided in the arguments
// for the DebianHost
func (h Debian) InstallPackage(names ...string) error {

	packages := ""
	for _, p := range names {
		packages += " " + p
	}

	return h.exec.Run(
		"bash",
		"-c",
		fmt.Sprintf("apt-get install -y %s", packages))
}

// EnableNTP service for the DebianHost
func (h Debian) EnableNTP() error {
	err := h.InstallPackage("ntp")
	err = h.exec.Run("bash", "-c", "systemctl enable --now ntp")
	return err
}

// SwapOff disables the swap.
func (h Debian) SwapOff() error {
	return h.exec.Run("bash", "-c", "swapoff -a")
}

func (h Debian) String() string {
	return "debian"
}

//PackagePresent checks if the package present on the host.
func (h Debian) PackagePresent(name string) bool {
	err := h.exec.Run("bash", "-c", fmt.Sprintf("dpkg -l | grep '%s'", name))
	return err == nil
}
