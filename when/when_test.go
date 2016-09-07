package when

import (
	"testing"
	"time"
)

func TestWhenAgo(t *testing.T) {
	parsed, err := Parse("5m ago")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if int(now.Sub(parsed).Minutes()) != 5 {
		t.Fatal("incorrect time")
	}
}

func TestWhenFixedTime(t *testing.T) {
	parsed, err := Parse("11:22:33 1/12/2006")
	if err != nil {
		t.Fatal(err)
	}

	if parsed.Second() != 33 {
		t.Fatal("incorrect second")
	}

	if parsed.Minute() != 22 {
		t.Fatal("incorrect minute")
	}

	if parsed.Hour() != 11 {
		t.Fatal("incorrect hour")
	}

	if parsed.Day() != 1 {
		t.Fatal("incorrect day")
	}

	if parsed.Month() != 12 {
		t.Fatal("incorrect month")
	}

	if parsed.Year() != 2006 {
		t.Fatal("incorrect year")
	}
}
