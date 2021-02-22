package cmdexec

var _ Executor = (*MockExecutor)(nil)

type MockExecutor struct {
	MockWithSudo      func() Executor
	MockRun           func(name string, args ...string) error
	MockRunWithStdout func(name string, args ...string) (string, error)
}

func (m *MockExecutor) WithSudo() Executor {
	return m.MockWithSudo()
}

func (m *MockExecutor) Run(name string, args ...string) error {
	return m.MockRun(name, args...)
}

func (m *MockExecutor) RunWithStdout(name string, args ...string) (string, error) {
	return m.MockRunWithStdout(name, args...)
}
