package critique

import "strings"

// EscalateUnresolved is a placeholder that would analyze previous vs current
// critique results and return emphasized directives for unresolved items.
func EscalateUnresolved(previous, current string) string {
    s := strings.TrimSpace(previous + "\n\n" + current)
    if s == "" { return "" }
    return "[CRITICAL — persisted] Fix unresolved items with imperative directives.\n\n" + s
}


