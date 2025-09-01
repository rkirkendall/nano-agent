package generate

import "strings"

// BuildEffectivePrompt joins the main prompt and non-empty fragments with blank lines
func BuildEffectivePrompt(main string, frags []string) string {
    parts := make([]string, 0, 1+len(frags))
    if strings.TrimSpace(main) != "" {
        parts = append(parts, main)
    }
    for _, f := range frags {
        f = strings.TrimSpace(f)
        if f != "" {
            parts = append(parts, f)
        }
    }
    return strings.Join(parts, "\n\n")
}


