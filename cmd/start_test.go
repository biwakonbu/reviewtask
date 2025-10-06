package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartCommand(t *testing.T) {
	t.Run("Valid task ID structure", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"start", "task-123"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		// May fail if task doesn't exist, but command structure should be valid
		if err != nil {
			// Check it's not a command structure error
			assert.NotContains(t, err.Error(), "accepts")
			assert.NotContains(t, err.Error(), "requires")
		}
	})
}

func TestStartCommandHelp(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"start", "--help"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "start <task-id>")
	assert.Contains(t, output, "Mark a task as \"doing\"")
	assert.Contains(t, output, "reviewtask start task-1")
}

func TestStartCommandIntegration(t *testing.T) {
	// Test that start command is properly registered
	cmd := NewRootCmd()
	startCmd, _, err := cmd.Find([]string{"start"})
	require.NoError(t, err)
	assert.NotNil(t, startCmd)
	assert.Equal(t, "start", startCmd.Name())
}
