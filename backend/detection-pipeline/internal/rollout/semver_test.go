package rollout

import "testing"

func TestSemverGE(t *testing.T) {
	if !SemverGE("1.2.0", "1.0.0") {
		t.Fatal("1.2 >= 1.0")
	}
	if SemverGE("0.9.0", "1.0.0") {
		t.Fatal("0.9 < 1.0")
	}
	if !SemverGE("1.0.0", "1.0.0") {
		t.Fatal("equal")
	}
	if !SemverGE("2.0.0", "1.99.99") {
		t.Fatal("major bump")
	}
}
