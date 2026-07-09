package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// MonthYearLayout — формат дат в API: месяц и год, например "07-2025".
const MonthYearLayout = "01-2006"

// MonthYear — дата с точностью до месяца. В API сериализуется строкой "MM-YYYY",
// в БД хранится как date (первое число месяца, UTC).
type MonthYear struct {
	time.Time
}

// NewMonthYear создаёт MonthYear, нормализованный к первому числу месяца в UTC.
func NewMonthYear(year int, month time.Month) MonthYear {
	return MonthYear{time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)}
}

// ParseMonthYear разбирает строку формата "MM-YYYY".
func ParseMonthYear(s string) (MonthYear, error) {
	t, err := time.Parse(MonthYearLayout, s)
	if err != nil {
		return MonthYear{}, fmt.Errorf("parse month-year %q: expected format MM-YYYY: %w", s, err)
	}
	return NewMonthYear(t.Year(), t.Month()), nil
}

func (m MonthYear) String() string {
	return m.Format(MonthYearLayout)
}

// MarshalJSON сериализует дату строкой "MM-YYYY".
func (m MonthYear) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

// UnmarshalJSON разбирает дату из строки "MM-YYYY".
func (m *MonthYear) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("month-year must be a string: %w", err)
	}

	parsed, err := ParseMonthYear(s)
	if err != nil {
		return err
	}

	*m = parsed
	return nil
}

// Scan реализует sql.Scanner: читает значение колонки date.
func (m *MonthYear) Scan(src any) error {
	t, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("scan MonthYear: unexpected type %T", src)
	}

	*m = NewMonthYear(t.Year(), t.Month())
	return nil
}

// Value реализует driver.Valuer: записывает значение в колонку date.
func (m MonthYear) Value() (driver.Value, error) {
	return m.Time, nil
}
