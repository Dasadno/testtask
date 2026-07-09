package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Dasadno/testtask/internal/models"
)

func TestParseMonthYear(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    time.Time
	}{
		{name: "valid", input: "07-2025", want: time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)},
		{name: "invalid month", input: "13-2025", wantErr: true},
		{name: "wrong format", input: "2025-07", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "not a date", input: "hello", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := models.ParseMonthYear(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseMonthYear(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMonthYear(%q) error = %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("ParseMonthYear(%q) = %v, want %v", tt.input, got.Time, tt.want)
			}
		})
	}
}

func TestMonthYear_JSONRoundTrip(t *testing.T) {
	src := models.NewMonthYear(2025, time.July)

	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `"07-2025"` {
		t.Errorf("Marshal() = %s, want %q", data, "07-2025")
	}

	var dst models.MonthYear
	if err := json.Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !dst.Equal(src.Time) {
		t.Errorf("round trip = %v, want %v", dst.Time, src.Time)
	}
}

func TestMonthYear_UnmarshalInvalid(t *testing.T) {
	var m models.MonthYear
	if err := json.Unmarshal([]byte(`"2025/07"`), &m); err == nil {
		t.Error("expected error for invalid format")
	}
	if err := json.Unmarshal([]byte(`42`), &m); err == nil {
		t.Error("expected error for non-string value")
	}
}

func TestMonthYear_Scan(t *testing.T) {
	var m models.MonthYear
	// БД возвращает date как time.Time; день и зона нормализуются к началу месяца UTC.
	src := time.Date(2025, time.July, 15, 10, 30, 0, 0, time.Local)
	if err := m.Scan(src); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	want := time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)
	if !m.Equal(want) {
		t.Errorf("Scan() = %v, want %v", m.Time, want)
	}

	if err := m.Scan("not a time"); err == nil {
		t.Error("expected error for non-time source")
	}
}
