package util

import "testing"

func TestParseDateRange(t *testing.T) {
	start, end, err := ParseDateRange("2015-01-02", "2015-12-30")
	if err != nil {
		t.Fatalf("ParseDateRange: %v", err)
	}
	if got := start.Format(dateLayout); got != "2015-01-02" {
		t.Errorf("start = %s, want 2015-01-02", got)
	}
	if got := end.Format(dateLayout); got != "2015-12-30" {
		t.Errorf("end = %s, want 2015-12-30", got)
	}
}

func TestParseDateRange_SameDay(t *testing.T) {
	if _, _, err := ParseDateRange("2015-01-02", "2015-01-02"); err != nil {
		t.Fatalf("ParseDateRange with equal start/end: %v", err)
	}
}

func TestParseDateRange_InvalidStart(t *testing.T) {
	if _, _, err := ParseDateRange("not-a-date", "2015-12-30"); err == nil {
		t.Fatal("expected error for invalid start")
	}
}

func TestParseDateRange_InvalidEnd(t *testing.T) {
	if _, _, err := ParseDateRange("2015-01-02", "not-a-date"); err == nil {
		t.Fatal("expected error for invalid end")
	}
}

func TestParseDateRange_EndBeforeStart(t *testing.T) {
	if _, _, err := ParseDateRange("2015-12-30", "2015-01-02"); err == nil {
		t.Fatal("expected error for end before start")
	}
}
