package generate

import (
    "testing"
)

func TestBuildEffectivePrompt(t *testing.T) {
    got := BuildEffectivePrompt("base", []string{"A", "", "B"})
    want := "base\n\nA\n\nB"
    if got != want {
        t.Fatalf("expected %q, got %q", want, got)
    }
}


