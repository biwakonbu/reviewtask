package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoldCommand(t *testing.T) {
	t.Run("Valid task ID structure", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"hold", "task-123"})

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

func TestHoldCommandHelp(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"hold", "--help"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "hold <task-id>")
	assert.Contains(t, output, "Mark a task as \"pending\"")
	assert.Contains(t, output, "reviewtask hold task-1")
}

func TestHoldCommandIntegration(t *testing.T) {
	// Test that hold command is properly registered
	cmd := NewRootCmd()
	holdCmd, _, err := cmd.Find([]string{"hold"})
	require.NoError(t, err)
	assert.NotNil(t, holdCmd)
	assert.Equal(t, "hold", holdCmd.Name())
}
