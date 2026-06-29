package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestFloatNumericRoundTrip(t *testing.T) {
	numeric, err := floatToNumeric(0.7)
	if err != nil {
		t.Fatal(err)
	}
	got, err := numericToFloat64(numeric)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0.7 {
		t.Fatalf("numeric round trip = %v, want 0.7", got)
	}
}

func TestNumericToFloat64Empty(t *testing.T) {
	got, err := numericToFloat64(pgtype.Numeric{})
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("empty numeric = %v, want 0", got)
	}
}
