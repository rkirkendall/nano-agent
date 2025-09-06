package critique

import "strings"

// BuildCritiqueInstruction returns the unified critique instruction text.
// It instructs the model to focus solely on actionable items to perform next,
// without referencing what has already been done in prior iterations.
func BuildCritiqueInstruction() string {
	var b strings.Builder
	b.WriteString("You are an expert image QA reviewer. Given the latest generated image (not the original), the original prompt, and any input reference images, return ONLY a single valid JSON object describing exactly what to KEEP and what to CHANGE next. No prose outside JSON.\n\n")
	b.WriteString("Use this exact schema:\n\n")
	b.WriteString("{")
	b.WriteString("\n  \"keep_notes\": [string, ...],")
	b.WriteString("\n  \"summary_keep\": [string, ...],")
	b.WriteString("\n  \"summary_change\": [string, ...],")
	b.WriteString("\n  \"edits\": [\n    {\n      \"id\": string,\n      \"target\": {\n        \"type\": \"object\"|\"region\"|\"global\",\n        \"label\": string,\n        \"bbox\": { \"x\": number, \"y\": number, \"w\": number, \"h\": number } | null,\n        \"points\": [ { \"x\": number, \"y\": number }, ... ] | null\n      },\n      \"priority\": \"CRITICAL\"|\"MAJOR\"|\"MINOR\",\n      \"instruction\": string,\n      \"rationale\": string,\n      \"done_when\": string\n    }\n  ]\n}")
	b.WriteString("\n\nRules:\n")
	b.WriteString("- JSON must be strictly valid and parseable; no markdown code fences.\n")
	b.WriteString("- No text outside the JSON object.\n")
	b.WriteString("- Coordinates normalized 0-1 relative to image width/height.\n")
	b.WriteString("- Max 8 edits; prioritize CRITICAL then MAJOR then MINOR.\n")
	return b.String()
}
