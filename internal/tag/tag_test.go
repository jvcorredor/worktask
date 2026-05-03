package tag

import "testing"

func TestNormalizeLowercases(t *testing.T) {
	got := Normalize("BUG")
	want := "bug"
	if got != want {
		t.Errorf("Normalize(%q) = %q, want %q", "BUG", got, want)
	}
}

func TestValidateAcceptsValidTag(t *testing.T) {
	if err := Validate("bug"); err != nil {
		t.Errorf("Validate(%q) = %v, want nil", "bug", err)
	}
}

func TestValidateRejectsInvalidChars(t *testing.T) {
	if err := Validate("bug!"); err == nil {
		t.Errorf("Validate(%q) = nil, want error", "bug!")
	}
}

func TestValidateRejectsTooLong(t *testing.T) {
	long := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := Validate(long); err == nil {
		t.Errorf("Validate(<41 chars>) = nil, want error")
	}
}

func TestValidateAcceptsHyphen(t *testing.T) {
	if err := Validate("high-priority"); err != nil {
		t.Errorf("Validate(%q) = %v, want nil", "high-priority", err)
	}
}

func TestDeduplicateRemovesDuplicates(t *testing.T) {
	got := Deduplicate([]string{"bug", "bug", "urgent"})
	want := []string{"bug", "urgent"}
	if len(got) != len(want) {
		t.Fatalf("Deduplicate len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Deduplicate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDeduplicatePreservesOrder(t *testing.T) {
	got := Deduplicate([]string{"urgent", "bug", "urgent"})
	want := []string{"urgent", "bug"}
	if len(got) != len(want) {
		t.Fatalf("Deduplicate len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Deduplicate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
