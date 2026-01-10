package exec

import (
	"testing"

	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/authfile"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/config"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSmartRunner_Creation(t *testing.T) {
	h := testutil.NewExtendedHarness(t)
	defer h.Close()
	
	// Mock dependencies
	vault := authfile.NewVault(h.TempDir)
	cfg := config.DefaultSPMConfig().Handoff
	
	runner := &Runner{} // Mock
	
	opts := SmartRunnerOptions{
		HandoffConfig: &cfg,
		Vault:         vault,
	}
	
	sr := NewSmartRunner(runner, opts)
	assert.NotNil(t, sr)
	assert.Equal(t, Running, sr.state)
}
