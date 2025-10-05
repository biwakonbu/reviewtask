package ui

import (
	"strings"
	"testing"
)

func TestHeader(t *testing.T) {
	result := Header("Test Header")
	if !strings.Contains(result, "Test Header") {
		t.Error("Header() should contain the title")
	}
}

func TestListItem(t *testing.T) {
	result := ListItem("Item 1")
	if !strings.Contains(result, "•") {
		t.Error("ListItem() should contain bullet point")
	}
	if !strings.Contains(result, "Item 1") {
		t.Error("ListItem() should contain the text")
	}
}

func TestKeyValue(t *testing.T) {
	result := KeyValue("Status", "complete")
	if !strings.Contains(result, "Status") {
		t.Error("KeyValue() should contain the key")
	}
	if !strings.Contains(result, "complete") {
		t.Error("KeyValue() should contain the value")
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		description string
		expectDesc  bool
	}{
		{
			name:        "with description",
			cmd:         "git status",
			description: "Show status",
			expectDesc:  true,
		},
		{
			name:        "without description",
			cmd:         "git push",
			description: "",
			expectDesc:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Command(tt.cmd, tt.description)
			if !strings.Contains(result, tt.cmd) {
				t.Errorf("Command() should contain command %q", tt.cmd)
			}
			if tt.expectDesc && tt.description != "" && !strings.Contains(result, tt.description) {
				t.Errorf("Command() should contain description %q", tt.description)
			}
		})
	}
}

func TestTable(t *testing.T) {
	headers := []string{"Name", "Status", "Priority"}
	rows := [][]string{
		{"Task 1", "TODO", "HIGH"},
		{"Task 2", "DONE", "LOW"},
	}

	result := Table(headers, rows)

	// Check headers are present
	for _, header := range headers {
		if !strings.Contains(result, header) {
			t.Errorf("Table() should contain header %q", header)
		}
	}

	// Check rows are present
	for _, row := range rows {
		for _, cell := range row {
			if !strings.Contains(result, cell) {
				t.Errorf("Table() should contain cell %q", cell)
			}
		}
	}

	// Check separator line
	if !strings.Contains(result, "─") {
		t.Error("Table() should contain separator line")
	}
}

func TestTableEmpty(t *testing.T) {
	result := Table([]string{}, [][]string{})
	if result != "" {
		t.Error("Table() with empty input should return empty string")
	}
}
