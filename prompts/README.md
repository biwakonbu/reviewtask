# Prompt Templates

This directory contains customizable prompt templates used by reviewtask for AI-powered task generation from PR review comments.

## Overview

Reviewtask uses external markdown templates with Go template syntax for generating AI prompts. This allows:
- Easy customization without recompilation
- Project-specific prompt tuning
- Version control tracking of prompt changes
- Rapid iteration and testing

## Template Files

### simple_task_generation.md

The main template for generating tasks from individual review comments.

**Purpose**: Convert a single review comment into actionable task items

**Variables**:
- `{{.LanguageInstruction}}` - Language preference from configuration (e.g., "Please respond in English")
- `{{.File}}` - File path where the comment was made
- `{{.Line}}` - Line number where the comment was placed
- `{{.Author}}` - Username of the comment author
- `{{.Comment}}` - The full text of the review comment

**Output Format**: JSON array of task objects with:
- `description` - Clear, actionable task description
- `priority` - Task priority level (critical/high/medium/low)

## Customization Guide

### Basic Customization

1. **Edit the template file directly**:
   ```bash
   vim prompts/simple_task_generation.md
   ```

2. **Test your changes**:
   ```bash
   # Debug mode to see rendered prompts without AI calls
   reviewtask debug prompt <PR-number> --profile v2
   ```

3. **Apply to real PR**:
   ```bash
   reviewtask fetch <PR-number>
   ```

### Language Customization

To customize for different languages, modify the instruction text:

```markdown
{{.LanguageInstruction}}Please generate tasks from the following comment.
```

The `LanguageInstruction` is set in `.pr-review/config.json`:
```json
{
  "ai_settings": {
    "user_language": "Japanese"
  }
}
```

### Priority Rules

Adjust priority detection by modifying the examples in the template:

```markdown
Examples of priority levels:
- critical: Security vulnerabilities, data loss, authentication issues
- high: Performance problems, broken functionality, memory leaks
- medium: Logic errors, missing validation, error handling
- low: Code style, documentation, minor improvements
```

### Few-Shot Examples

The template includes few-shot examples to guide AI behavior:

```json
[
  {
    "description": "Add nil check for user object before accessing properties",
    "priority": "high"
  },
  {
    "description": "Fix typo in variable name 'recieve' to 'receive'",
    "priority": "low"
  }
]
```

Modify these examples to match your team's task generation preferences.

## Best Practices

### 1. Keep Templates Focused

Each template should have a single, clear purpose. Don't try to handle multiple scenarios in one template.

### 2. Use Clear Examples

Provide 2-3 representative examples that show the desired output format and content style.

### 3. Specify Priority Criteria

Include clear guidelines for priority assignment that match your team's workflow.

### 4. Test Changes Thoroughly

Always test template changes with `reviewtask debug prompt` before processing real PRs:

```bash
# Test with different PR sizes and comment styles
reviewtask debug prompt 123 --profile v2
reviewtask debug prompt 456 --profile v2
```

### 5. Version Control Templates

Track template changes in git to understand evolution and rollback if needed:

```bash
git add prompts/
git commit -m "Adjust task priority rules for security-focused reviews"
```

## Template Variables Reference

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.LanguageInstruction` | string | Language directive from config | "Please respond in Japanese" |
| `.File` | string | File path from comment location | "internal/storage/manager.go" |
| `.Line` | int | Line number in file | 42 |
| `.Author` | string | GitHub username of commenter | "alice" |
| `.Comment` | string | Full comment body text | "This needs error handling" |

## Advanced Usage

### Creating Custom Templates

1. **Create new template file**:
   ```bash
   touch prompts/custom_security_review.md
   ```

2. **Define template structure**:
   ```markdown
   # Security Review Task Generation

   {{.LanguageInstruction}}Analyze this security-related comment and generate remediation tasks.

   Comment: {{.Comment}}
   Location: {{.File}}:{{.Line}}

   Generate tasks focusing on security implications...
   ```

3. **Reference in code** (requires code modification):
   ```go
   analyzer.loadPromptTemplate("custom_security_review.md", data)
   ```

### Template Testing

Use golden tests to ensure template stability:

```bash
# Run golden tests
go test ./internal/ai -run Golden

# Update golden files when intentionally changing templates
UPDATE_GOLDEN=1 go test ./internal/ai -run Golden
```

### Performance Considerations

- **Template Size**: Keep templates under 2KB for optimal performance
- **Example Count**: 2-3 examples are usually sufficient
- **Variable Usage**: Use all provided variables for better context
- **JSON Format**: Always use proper JSON in examples with quotes and commas

## Troubleshooting

### Template Not Loading

Check file exists and has correct permissions:
```bash
ls -la prompts/simple_task_generation.md
```

### Malformed JSON Output

Ensure examples in template use valid JSON:
```bash
# Validate JSON examples
cat prompts/simple_task_generation.md | grep -A 5 '```json' | jq .
```

### Inconsistent Task Generation

Review template examples and ensure they match desired output:
```bash
# Compare actual vs expected output
reviewtask debug prompt <PR> --verbose
```

### Language Not Applying

Verify configuration:
```bash
cat .pr-review/config.json | jq '.ai_settings.user_language'
```

## Migration from Hardcoded Prompts

If upgrading from a version with hardcoded prompts:

1. **Backup existing configuration**:
   ```bash
   cp -r .pr-review .pr-review.backup
   ```

2. **Initialize templates**:
   ```bash
   reviewtask init
   ```

3. **Customize templates** as needed

4. **Test with debug commands** before production use

## Contributing

When contributing template improvements:

1. Test changes with multiple PR types
2. Update golden tests if output format changes
3. Document any new variables or patterns
4. Include examples in pull requests

For more information, see the [Architecture Documentation](../docs/architecture.md#prompt-template-system).