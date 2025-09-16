package github

import (
	"strings"
	"testing"
)

func TestRemoveCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no code blocks",
			input:    "This is a simple comment without any code.",
			expected: "This is a simple comment without any code.",
		},
		{
			name: "fenced code block",
			input: `Please fix this issue:

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

Thanks!`,
			expected: `Please fix this issue:

[code block removed]

Thanks!`,
		},
		{
			name:     "inline code",
			input:    `Please update the ` + "`variable`" + ` name to be more descriptive.`,
			expected: `Please update the [code removed] name to be more descriptive.`,
		},
		{
			name: "indented code block",
			input: `Here's the problem:

    func broken() {
        return nil
    }

Fix it please.`,
			expected: `Here's the problem:

[code block removed]

Fix it please.`,
		},
		{
			name: "mixed code types",
			input: `Issue with ` + "`getData()`" + ` function:

` + "```javascript" + `
const data = getData();
if (!data) {
    throw new Error('No data');
}
` + "```" + `

Also this indented code:

    const x = 5;
    const y = 10;

Please fix both issues.`,
			expected: `Issue with [code removed] function:

[code block removed]

Also this indented code:

[code block removed]

Please fix both issues.`,
		},
		{
			name: "multiple consecutive code blocks",
			input: `First:
` + "```go" + `
func a() {}
` + "```" + `

Then:
` + "```go" + `
func b() {}
` + "```" + `

Done.`,
			expected: `First:
[code block removed]

Then:
[code block removed]

Done.`,
		},
		{
			name: "code block without language",
			input: `Example:

` + "```" + `
some code here
more code
` + "```" + `

End.`,
			expected: `Example:

[code block removed]

End.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeCodeBlocks(tt.input)
			if result != tt.expected {
				t.Errorf("removeCodeBlocks() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRemoveCodeBlocksPreservesStructure(t *testing.T) {
	input := `# Main Issue

This is the main problem description.

## Code Problems

The function ` + "`calculateTotal`" + ` has issues:

` + "```typescript" + `
function calculateTotal(items: Item[]): number {
    let total = 0;
    for (const item of items) {
        total += item.price;
    }
    return total;
}
` + "```" + `

**Issues:**
1. No null checking
2. No error handling

Please fix these.

## Additional Notes

Also check this pattern:

    if (user) {
        user.save();
    }

Thanks!`

	result := removeCodeBlocks(input)

	// Should preserve structure
	if !strings.Contains(result, "# Main Issue") {
		t.Error("Should preserve markdown headers")
	}

	if !strings.Contains(result, "**Issues:**") {
		t.Error("Should preserve bold formatting")
	}

	if !strings.Contains(result, "1. No null checking") {
		t.Error("Should preserve list formatting")
	}

	// Should remove code
	if strings.Contains(result, "calculateTotal") {
		t.Error("Should remove code block content")
	}

	if strings.Contains(result, "user.save()") {
		t.Error("Should remove indented code block")
	}

	// Should have code removal markers
	if !strings.Contains(result, "[code removed]") {
		t.Error("Should have inline code removal markers")
	}

	if !strings.Contains(result, "[code block removed]") {
		t.Error("Should have code block removal markers")
	}
}
