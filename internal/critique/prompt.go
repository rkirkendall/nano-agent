package critique

import "strings"

// BuildCritiqueInstruction returns the unified critique instruction text.
// It instructs the model to focus solely on actionable items to perform next,
// without referencing what has already been done in prior iterations.
func BuildCritiqueInstruction() string {
	var b strings.Builder
	b.WriteString("You are an expert image forensics and quality reviewer. Given the latest generated image (not the original) and the original text prompt, write a concise critique that reflects what is still missing or incorrect right now.\n\n")
	b.WriteString("Be iteration-aware: compare the current image against the original prompt and the likely prior intent, and only call out items that remain unresolved. If an item appears fixed, do not restate it.\n\n")
	b.WriteString("- Identify visual artifacts, anatomical errors, texture inconsistencies, lighting/shadow mismatches, reflections, perspective errors, or other realism issues.\n")
	b.WriteString("- Point out mismatches between the image and the prompt.\n")
	b.WriteString("- Provide concrete, targeted suggestions to fix the issues in the next image iteration.\n\n")
	b.WriteString("Return plain text.")
	return b.String()
}
