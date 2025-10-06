package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoneCommand(t *testing.T) {
	t.Run("Valid task ID structure", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"done", "task-123"})

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

func TestDoneCommandHelp(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"done", "--help"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "done <task-id>")
	assert.Contains(t, output, "automated workflow")
	assert.Contains(t, output, "reviewtask done task-1")
}

func TestDoneCommandIntegration(t *testing.T) {
	// Test that done command is properly registered
	cmd := NewRootCmd()
	doneCmd, _, err := cmd.Find([]string{"done"})
	require.NoError(t, err)
	assert.NotNil(t, doneCmd)
	assert.Equal(t, "done", doneCmd.Name())
}
