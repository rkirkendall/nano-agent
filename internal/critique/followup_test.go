package critique

import "testing"

func TestEscalateUnresolved(t *testing.T) {
    prev := "- [ ] fix eyes symmetry\n- [ ] remove haloing around head"
    cur := "eyes still asymmetric; halo remains"
    out := EscalateUnresolved(prev, cur)
    if out == "" {
        t.Fatal("expected non-empty escalation output")
    }
}


