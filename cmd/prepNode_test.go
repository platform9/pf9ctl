package cmd

import (
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/stretchr/testify/assert"
	"testing"
)

//Local executor test case
func TestGetExecutor(t *testing.T) {
	var exec cmdexec.LocalExecutor
	var TestErr error = nil

	t.Run("LocalExecutorTest", func(t *testing.T) {
		executor, err := getExecutor()
		if err != nil {
			t.Errorf("Error occured : %s", err)
		}
		assert.Equal(t, TestErr, err)
		assert.Equal(t, exec, executor)
	})
}
