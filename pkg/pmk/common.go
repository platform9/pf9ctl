package pmk

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"github.com/platform9/pf9ctl/pkg/log"
)

func setupNode(hostOS string) (err error) {
	log.Info.Println("Received a call to setup the node")

	if err := swapOff(); err != nil {
		return err
	}

	if err := handlePF9UserGroup(); err != nil {
		return err
	}

	switch hostOS {
	case "redhat":
		err = redhatCentosPackageInstall()
		if err != nil {
			return
		}
		err = ntpInstallActivateRedhatCentos()

	case "debian":
		err = ubuntuPackageInstall()
		if err != nil {
			return
		}
		err = ntpInstallActivateUbuntu()

	default:
		err = fmt.Errorf("Invalid Host: %s", hostOS)
	}

	return
}

func handlePF9UserGroup() error {

	err := os.MkdirAll("/opt/pf9", 0755)
	if err != nil {
		return err
	}

	err = createPF9Group("pf9group")
	if err != nil {
		return err
	}

	return createPF9User("pf9")
}

func createPF9User(name string) error {
	log.Info.Println("Received a call to create PF9User")

	_, err := user.Lookup(name)
	if err != nil {

		if _, ok := err.(user.UnknownUserError); !ok {
			return err
		}

		log.Info.Println("User not present, creating it")
		if _, err := exec.Command("bash", "-c", "sudo useradd -g pf9group -d '/opt/pf9/home' -s '/bin/bash'  pf9").Output(); err != nil {
			return err
		}
	}

	return nil
}

func createPF9Group(name string) error {
	log.Info.Println("Received a call to create Pf9 group")

	_, err := user.LookupGroup(name)
	if err != nil {

		if _, ok := err.(user.UnknownGroupError); !ok {
			return fmt.Errorf("Unable to crate a pf9group: %s", err.Error())
		}

		cmd := fmt.Sprintf("sudo groupadd %s", name)
		if _, err := exec.Command(
			"bash", "-c", cmd).Output(); err != nil {
			return err
		}

	}

	return nil
}

func ubuntuPackageInstall() error {
	log.Info.Println("Received a call to perform ubuntu package install")

	_, err := exec.Command("bash", "-c", "sudo apt-get update ").Output()
	_, err = exec.Command("bash", "-c", "sudo apt-get install curl uuid-runtime software-properties-common logrotate -y").Output()
	return err
}

func redhatCentosPackageInstall() error {
	log.Info.Println("Received a call to perform redhat package install")

	_, err := exec.Command("bash", "-c", "sudo yum install libselinux-python -y").Output()
	return err
}

func ntpInstallActivateUbuntu() error {
	log.Info.Println("Received a call to install ntp")

	_, err := exec.Command("bash", "-c", "sudo apt-get install ntp -y").Output()
	if err != nil {
		fmt.Errorf("ntp package installation failed: %s", err.Error())
	}

	log.Debug.Println("ntpd installation completed successfully")
	_, err = exec.Command("bash", "-c", "sudo systemctl enable --now ntp").Output()
	if err != nil {
		fmt.Errorf("ntp startup failed: %s", err.Error())
	}

	return nil
}
func ntpInstallActivateRedhatCentos() error {
	_, err := exec.Command("bash", "-c", "sudo yum install ntp -y").Output()
	if err != nil {
		fmt.Errorf("ntp package installation failed: %s", err.Error())
	}

	log.Debug.Println("ntpd installation completed successfully")
	_, err = exec.Command("bash", "-c", "sudo systemctl enable --now ntpd").Output()
	if err != nil {
		fmt.Errorf("ntp startup failed: %s", err.Error())
	}

	return nil
}

func swapOff() error {
	log.Info.Println("Received call to Disabling swap")

	_, err := exec.Command("bash", "-c", "swapoff -a").Output()
	return err
}
