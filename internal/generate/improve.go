package generate

import (
	"strings"
)

// BuildImprovementPrompt creates a single improvement instruction that
// references the original prompt and appends the latest critique.
// The critique should represent only actionable next steps.
func BuildImprovementPrompt(originalPrompt, critique string) string {
	orig := strings.TrimSpace(originalPrompt)
	crt := strings.TrimSpace(critique)
	var b strings.Builder
	b.WriteString("This image was generated with the following original prompt:\n\n")
	if orig != "" {
		b.WriteString(orig)
	} else {
		b.WriteString("(no original prompt provided)")
	}
	b.WriteString("\n\nNow apply the critique below to improve the image. Prioritize items tagged [CRITICAL — persisted] first, then [MAJOR], then [MINOR]. Use decisive, localized fixes and avoid regressions on items marked done. Then implement the 'Targeted actions to apply now' if present.\n\nCritique follows:\n\n")
	if crt != "" {
		b.WriteString(crt)
	} else {
		b.WriteString("(no critique provided)")
	}
	return b.String()
}

// BuildImprovementPromptWithActions augments the improvement instruction with a
// structured JSON actions object that enumerates precise edits to perform.
// The JSON must be included verbatim so the model can follow exact steps.
func BuildImprovementPromptWithActions(originalPrompt, critique, actionsJSON string) string {
	orig := strings.TrimSpace(originalPrompt)
	crt := strings.TrimSpace(critique)
	aj := strings.TrimSpace(actionsJSON)
	var b strings.Builder
	b.WriteString("This image was generated with the following original prompt:\n\n")
	if orig != "" {
		b.WriteString(orig)
	} else {
		b.WriteString("(no original prompt provided)")
	}
	b.WriteString("\n\nApply the critique below and follow the JSON 'actions' exactly. Prioritize items tagged [CRITICAL — persisted] first, then [MAJOR], then [MINOR]. Avoid regressions on previously fixed items.\n\nCritique follows:\n\n")
	if crt != "" {
		b.WriteString(crt)
	} else {
		b.WriteString("(no critique provided)")
	}
	if aj != "" {
		b.WriteString("\n\nActions (JSON):\n")
		b.WriteString(aj)
	}
	return b.String()
}
