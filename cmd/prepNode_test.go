package cmd

import (
	"testing"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/stretchr/testify/assert"
)

var proxy_url = "127.0.0.1:3128"

//Local executor test case
func TestGetExecutor(t *testing.T) {
	exec := cmdexec.LocalExecutor{ProxyUrl: proxy_url}
	var TestErr error = nil

	t.Run("LocalExecutorTest", func(t *testing.T) {
		executor, err := getExecutor(proxy_url)
		if err != nil {
			t.Errorf("Error occured : %s", err)
		}
		assert.Equal(t, TestErr, err)
		assert.Equal(t, exec, executor)
	})
}
