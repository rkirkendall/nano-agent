package critique

import "strings"

// BuildCritiqueInstruction returns the unified critique instruction text.
// It instructs the model to focus solely on actionable items to perform next,
// without referencing what has already been done in prior iterations.
func BuildCritiqueInstruction() string {
	var b strings.Builder
	b.WriteString("You are an expert image forensics and quality reviewer. Given the provided image and the original text prompt, write a concise critique.\n\n")
	b.WriteString("Focus only on actionable items that should be done in the next generation. Do not reference, list, or discuss items that were already completed previously.\n\n")
	b.WriteString("- Identify visual artifacts, anatomical errors, texture inconsistencies, lighting/shadow mismatches, reflections, perspective errors, or other realism issues.\n")
	b.WriteString("- Point out mismatches between the image and the prompt.\n")
	b.WriteString("- Provide concrete, targeted suggestions to fix the issues in the next image iteration.\n\n")
	b.WriteString("Return plain text.")
	return b.String()
}
