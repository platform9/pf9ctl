package cmdexec

import (
	"testing"

	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/stretchr/testify/assert"
)

func TestGetExecutor(t *testing.T) {
	proxyURL := "127.0.0.1"

	localExecutor := LocalExecutor{ProxyUrl: proxyURL}

	// Test local executor
	executor, err := GetExecutor(proxyURL, objects.NodeConfig{})
	assert.Equal(t, nil, err)
	assert.Equal(t, localExecutor, executor)
}

// The installer command line as RunWithProgressStages logs it (PCD-9384). The
// old string-cutting remover kept everything after the first space of a
// quoted password and never touched the proxy URL.
func TestConfidentialInfoRemover_MasksInstallerCommand(t *testing.T) {
	cmd := ` "https_proxy=http://pf9:pr0xy@10.0.0.1:3128" "bash" "-c" "/root/pf9/installer.sh --proxy http://pf9:pr0xy@10.0.0.1:3128 --skip-os-check --no-project --controller=du.example.com --username=admin@example.com --password='S3 cret' --insecure"`

	got := ConfidentialInfoRemover(cmd)

	assert.Equal(t, ` "https_proxy=http://*****:*****@10.0.0.1:3128" "bash" "-c" "/root/pf9/installer.sh --proxy http://*****:*****@10.0.0.1:3128 --skip-os-check --no-project --controller=du.example.com --username=admin@example.com --password=***** --insecure"`, got)
}

// Negative case: a command with nothing secret is logged as it is.
func TestConfidentialInfoRemover_LeavesPlainCommandAlone(t *testing.T) {
	cmd := ` "bash" "-c" "chmod +x /root/pf9/installer.sh"`
	assert.Equal(t, cmd, ConfidentialInfoRemover(cmd))
}
