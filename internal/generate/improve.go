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
