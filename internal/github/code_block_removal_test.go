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

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

Thanks!`,
		},
		{
			name:     "inline code",
			input:    `Please update the ` + "`variable`" + ` name to be more descriptive.`,
			expected: `Please update the ` + "`variable`" + ` name to be more descriptive.`,
		},
		{
			name: "indented code block",
			input: `Here's the problem:

    func broken() {
        return nil
    }

Fix it please.`,
			expected: `Here's the problem:

    func broken() {
        return nil
    }

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
			expected: `Issue with ` + "`getData()`" + ` function:

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
` + "```go" + `
func a() {}
` + "```" + `

Then:
` + "```go" + `
func b() {}
` + "```" + `

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

` + "```" + `
some code here
more code
` + "```" + `

End.`,
		},
		{
			name: "GitHub suggestion block",
			input: `Please fix this:

<!-- suggestion_start -->
func improved() {
    return "better"
}
<!-- suggestion_end -->

Thanks!`,
			expected: `Please fix this:

Thanks!`,
		},
		{
			name: "GitHub committable suggestion",
			input: `<details>
<summary>📝 Committable suggestion</summary>

` + "```go" + `
func newFunction() {
    // implementation
}
` + "```" + `

</details>

Please review this change.`,
			expected: `Please review this change.`,
		},
		{
			name: "GitHub AI prompt section",
			input: `Review comment here.

<details>
<summary>🤖 Prompt for AI Agents</summary>

This is a prompt for AI agents with detailed instructions.

</details>

End of comment.`,
			expected: `Review comment here.

<details>
<summary>🤖 Prompt for AI Agents</summary>

This is a prompt for AI agents with detailed instructions.

</details>

End of comment.`,
		},
		{
			name: "CodeRabbit actionable comments header removal",
			input: `**Actionable comments posted: 5**

> [!CAUTION]
> Some important feedback here

**Issues found:**
1. Missing error handling
2. Performance concern

Please fix these issues.`,
			expected: `> [!CAUTION]
> Some important feedback here

**Issues found:**
1. Missing error handling
2. Performance concern

Please fix these issues.`,
		},
		{
			name: "GitHub fingerprinting",
			input: `Good suggestion!

<!-- fingerprinting:phantom:medusa:chinchilla -->

Please implement.`,
			expected: `Good suggestion!

Please implement.`,
		},
		{
			name: "mixed GitHub features and code",
			input: `Review feedback:

<!-- suggestion_start -->
` + "```diff" + `
- old code
+ new code
` + "```" + `
<!-- suggestion_end -->

<details>
<summary>📝 Committable suggestion</summary>
More code here
</details>

<!-- fingerprinting:test -->`,
			expected: `Review feedback:`,
		},
		{
			name: "HTML escaped GitHub features",
			input: `Review feedback:

\u003c!-- suggestion_start --\u003e
` + "```go" + `
func test() {}
` + "```" + `
\u003c!-- suggestion_end --\u003e

\u003cdetails\u003e
\u003csummary\u003e📝 Committable suggestion\u003c/summary\u003e
More code here
\u003c/details\u003e

\u003c!-- fingerprinting:phantom:medusa:chinchilla --\u003e`,
			expected: `Review feedback:`,
		},
		{
			name: "CodeRabbit detailed review body",
			input: `**Actionable comments posted: 11**

\u003e [!CAUTION]
\u003e Some comments are outside the diff and can't be posted inline due to platform limitations.
\u003e
\u003e
\u003e
\u003e \u003cdetails\u003e
\u003e \u003csummary\u003e⚠️ Outside diff range comments (2)\u003c/summary\u003e\u003cblockquote\u003e
\u003e
\u003e \u003cdetails\u003e
\u003e \u003csummary\u003einternal/config/config.go (1)\u003c/summary\u003e\u003cblockquote\u003e
\u003e
\u003e ` + "`257-273`" + `: **Backfilling new boolean flags is brittle; missing fields may silently default to false**
\u003e
\u003e Because JSON unmarshaling sets absent bools to false, non-"old" configs that omit the new fields will disable features by default. This violates "predictable defaults".
\u003e
\u003e Recommendation:
\u003e - Represent new booleans as pointers (*bool) to distinguish "unset" from explicit false, or
\u003e - Detect field presence by unmarshaling into a temporary map[string]any and backfilling only when the key is absent.
\u003e I can draft a safe backfill helper if you want.
\u003e
\u003e \u003c/blockquote\u003e\u003c/details\u003e
\u003e
\u003e \u003c/blockquote\u003e\u003c/details\u003e

\u003cdetails\u003e
\u003csummary\u003e🧹 Nitpick comments (16)\u003c/summary\u003e\u003cblockquote\u003e

\u003cdetails\u003e
\u003csummary\u003einternal/ai/enhanced_json_recovery.go (2)\u003c/summary\u003e\u003cblockquote\u003e

` + "`16-27`" + `: **Config knobs defined but unused (MaxRecoveryAttempts, PartialThreshold, LogTruncatedResponses)**

These defaults are never referenced in the strategies. Either wire them into behavior (attempt loops/threshold gating/logging) or remove to avoid misleading configuration.

---

` + "`123-156`" + `: **Truncation completion may key off braces inside strings**

Using strings.LastIndex on "{" ignores whether it's inside quotes. This can yield nonsensical completions. Consider a simple lexer to find the last unquoted '{'.

\u003c/blockquote\u003e\u003c/details\u003e

\u003c/blockquote\u003e\u003c/details\u003e

\u003cdetails\u003e
\u003csummary\u003e📜 Review details\u003c/summary\u003e

**Configuration used**: CodeRabbit UI

**Review profile**: CHILL

**Plan**: Pro

\u003cdetails\u003e
\u003csummary\u003e📥 Commits\u003c/summary\u003e

Reviewing files that changed from the base of the PR and between 538885d8d2227ae5395f575c2f378e389b86e003 and 23eaa7af3c0012f3de03a1e805eff75c39fd31da.

\u003c/details\u003e

\u003c/details\u003e`,
			expected: `> [!CAUTION]
> Some comments are outside the diff and can't be posted inline due to platform limitations.
>
>
>
> <details>
> <summary>⚠️ Outside diff range comments (2)</summary><blockquote>
>
> <details>
> <summary>internal/config/config.go (1)</summary><blockquote>
>
> ` + "`257-273`" + `: **Backfilling new boolean flags is brittle; missing fields may silently default to false**
>
> Because JSON unmarshaling sets absent bools to false, non-"old" configs that omit the new fields will disable features by default. This violates "predictable defaults".
>
> Recommendation:
> - Represent new booleans as pointers (*bool) to distinguish "unset" from explicit false, or
> - Detect field presence by unmarshaling into a temporary map[string]any and backfilling only when the key is absent.
> I can draft a safe backfill helper if you want.
>
> </blockquote></details>
>
> </blockquote></details>

<details>
<summary>🧹 Nitpick comments (16)</summary><blockquote>

<details>
<summary>internal/ai/enhanced_json_recovery.go (2)</summary><blockquote>

` + "`16-27`" + `: **Config knobs defined but unused (MaxRecoveryAttempts, PartialThreshold, LogTruncatedResponses)**

These defaults are never referenced in the strategies. Either wire them into behavior (attempt loops/threshold gating/logging) or remove to avoid misleading configuration.

---

` + "`123-156`" + `: **Truncation completion may key off braces inside strings**

Using strings.LastIndex on "{" ignores whether it's inside quotes. This can yield nonsensical completions. Consider a simple lexer to find the last unquoted '{'.

</blockquote></details>

</blockquote></details>

<details>
<summary>📜 Review details</summary>

**Configuration used**: CodeRabbit UI

**Review profile**: CHILL

**Plan**: Pro

<details>
<summary>📥 Commits</summary>

Reviewing files that changed from the base of the PR and between 538885d8d2227ae5395f575c2f378e389b86e003 and 23eaa7af3c0012f3de03a1e805eff75c39fd31da.

</details>

</details>`,
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

	// Should preserve code
	if !strings.Contains(result, "calculateTotal") {
		t.Error("Should preserve inline code")
	}

	if !strings.Contains(result, "user.save()") {
		t.Error("Should preserve indented code block")
	}

	if !strings.Contains(result, "function calculateTotal") {
		t.Error("Should preserve fenced code block content")
	}
}

// Test AI Prompt preservation functionality
func TestRemoveCodeBlocks_AIPromptPreservation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		shouldContainPrompt bool
		shouldRemoveSuggestions bool
	}{
		{
			name: "AI Prompt block with HTML entities should be preserved",
			input: `_🛠️ Refactor suggestion_

**Fix error handling**

Current code doesn't handle errors properly.

\u003cdetails\u003e
\u003csummary\u003e🤖 Prompt for AI Agents\u003c/summary\u003e

\u003cpre\u003e\u003ccode\u003e
In internal/ai/analyzer.go around lines 1036 to 1051, fix the error handling.
\u003c/code\u003e\u003c/pre\u003e

\u003c/details\u003e

\u003c!-- This is an auto-generated comment by CodeRabbit --\u003e`,
			shouldContainPrompt: true,
			shouldRemoveSuggestions: false,
		},
		{
			name: "AI Prompt block with normal HTML should be preserved",
			input: `_🛠️ Refactor suggestion_

**Fix error handling**

<details>
<summary>🤖 Prompt for AI Agents</summary>

` + "```" + `
In internal/ai/analyzer.go around lines 1036 to 1051, fix the error handling.
` + "```" + `

</details>

<!-- This is an auto-generated comment by CodeRabbit -->`,
			shouldContainPrompt: true,
			shouldRemoveSuggestions: false,
		},
		{
			name: "Suggestion blocks should be removed",
			input: `_🛠️ Refactor suggestion_

**Fix error handling**

` + "```diff" + `
- old code
+ new code
` + "```" + `

<!-- suggestion_start -->
<details>
<summary>📝 Committable suggestion</summary>

` + "```suggestion" + `
new code
` + "```" + `

</details>
<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

` + "```" + `
Fix the error handling here.
` + "```" + `

</details>`,
			shouldContainPrompt: true,
			shouldRemoveSuggestions: true,
		},
		{
			name: "No AI Prompt - only suggestion removal",
			input: `_🛠️ Refactor suggestion_

**Fix error handling**

<!-- suggestion_start -->
<details>
<summary>📝 Committable suggestion</summary>

` + "```suggestion" + `
new code
` + "```" + `

</details>
<!-- suggestion_end -->`,
			shouldContainPrompt: false,
			shouldRemoveSuggestions: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeCodeBlocks(tt.input)

			if tt.shouldContainPrompt {
				if !strings.Contains(result, "🤖 Prompt for AI Agents") {
					t.Errorf("Expected AI Prompt to be preserved, but it was removed. Result: %s", result)
				}
			}

			if tt.shouldRemoveSuggestions {
				if strings.Contains(result, "suggestion_start") || strings.Contains(result, "suggestion_end") {
					t.Errorf("Expected suggestion blocks to be removed, but they were preserved. Result: %s", result)
				}
				if strings.Contains(result, "📝 Committable suggestion") {
					t.Errorf("Expected committable suggestions to be removed, but they were preserved. Result: %s", result)
				}
			}

			// Verify HTML entities are unescaped
			if strings.Contains(result, "\\u003c") || strings.Contains(result, "\\u003e") {
				t.Errorf("HTML entities should be unescaped. Result: %s", result)
			}
		})
	}
}

// Test HTML entity unescaping
func TestProcessAIPromptAndSuggestions_HTMLUnescaping(t *testing.T) {
	input := `\u003cdetails\u003e
\u003csummary\u003e🤖 Prompt for AI Agents\u003c/summary\u003e

\u003cpre\u003e\u003ccode\u003e
Fix the error handling
\u003c/code\u003e\u003c/pre\u003e

\u003c/details\u003e`

	result := processAIPromptAndSuggestions(input)

	// Should unescape HTML entities
	if strings.Contains(result, "\\u003c") || strings.Contains(result, "\\u003e") {
		t.Errorf("HTML entities should be unescaped. Got: %s", result)
	}

	// Should contain proper HTML tags
	if !strings.Contains(result, "<details>") || !strings.Contains(result, "</details>") {
		t.Errorf("Should contain proper HTML tags after unescaping. Got: %s", result)
	}
}
