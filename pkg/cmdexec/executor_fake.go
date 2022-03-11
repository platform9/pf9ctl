package cmdexec

var _ Executor = (*MockExecutor)(nil)

type MockExecutor struct {
	MockRun            func(name string, args ...string) error
	MockRunWithStdout  func(name string, args ...string) (string, error)
	MockRunCommandWait func(name string) string
}

func (m *MockExecutor) Run(name string, args ...string) error {
	return m.MockRun(name, args...)
}

func (m *MockExecutor) RunWithStdout(name string, args ...string) (string, error) {
	return m.MockRunWithStdout(name, args...)
}

func (m *MockExecutor) RunCommandWait(name string) string {
	return m.RunCommandWait(name)
}
