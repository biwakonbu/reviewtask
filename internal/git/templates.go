package git

import (
	"bytes"
	"text/template"
)

// CommitMessageTemplate defines a template for commit messages
type CommitMessageTemplate struct {
	Template *template.Template
	Language string
}

// DefaultTemplates contains default commit message templates for different languages
var DefaultTemplates = map[string]string{
	"en": `{{.TaskSummary}}

{{if .ReviewCommentURL}}Review Comment: {{.ReviewCommentURL}}

{{end}}{{if .OriginalComment}}Original Comment:
{{range $line := splitLines .OriginalComment}}> {{$line}}
{{end}}
{{end}}{{if .Changes}}Changes:
{{range .Changes}}- {{.}}
{{end}}
{{end}}{{if .PRNumber}}PR: #{{.PRNumber}}{{end}}`,

	"ja": `{{.TaskSummary}}

{{if .ReviewCommentURL}}レビューコメント: {{.ReviewCommentURL}}

{{end}}{{if .OriginalComment}}元のコメント:
{{range $line := splitLines .OriginalComment}}> {{$line}}
{{end}}
{{end}}{{if .Changes}}変更内容:
{{range .Changes}}- {{.}}
{{end}}
{{end}}{{if .PRNumber}}PR: #{{.PRNumber}}{{end}}`,
}

// NewCommitMessageTemplate creates a new commit message template for the specified language
func NewCommitMessageTemplate(language string) (*CommitMessageTemplate, error) {
	templateStr, ok := DefaultTemplates[language]
	if !ok {
		// Default to English if language not found
		templateStr = DefaultTemplates["en"]
		language = "en"
	}

	tmpl, err := template.New("commit").Funcs(template.FuncMap{
		"splitLines": splitLines,
	}).Parse(templateStr)
	if err != nil {
		return nil, err
	}

	return &CommitMessageTemplate{
		Template: tmpl,
		Language: language,
	}, nil
}

// Render renders the commit message template with the provided data
func (t *CommitMessageTemplate) Render(data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := t.Template.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// splitLines is a template helper function to split text into lines
func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}

	var lines []string
	var currentLine []rune

	for _, r := range text {
		if r == '\n' {
			lines = append(lines, string(currentLine))
			currentLine = currentLine[:0]
		} else {
			currentLine = append(currentLine, r)
		}
	}

	// Add the last line if it's not empty
	if len(currentLine) > 0 {
		lines = append(lines, string(currentLine))
	}

	return lines
}
