package clients

import "os/exec"

type Executor interface {
	Run(name string, args ...string) error
	RunWithStdout(name string, args ...string) (string, error)
}

type ExecutorImpl struct{}

func (c ExecutorImpl) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func (c ExecutorImpl) RunWithStdout(name string, args ...string) (string, error) {
	byt, err := exec.Command(name, args...).Output()
	return string(byt), err
}
