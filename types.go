package duitku

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// StringOrInt holds a JSON value that Duitku may send either as a quoted
// string ("40000") or as a bare number (40000).
//
// The Duitku docs are internally inconsistent about this: the response tables
// type `amount` as an integer while the JSON examples quote it. StringOrInt
// accepts both, so a response never fails to decode because of that
// discrepancy. It marshals back as a bare number when the value is an integer
// literal and as a string otherwise.
type StringOrInt string

// String returns the underlying value as a plain string.
func (s StringOrInt) String() string {
	return string(s)
}

// Int converts the value to an int, reporting an error when it is empty or not
// numeric.
func (s StringOrInt) Int() (int, error) {
	if s == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(string(s))
	if err != nil {
		return 0, fmt.Errorf("duitku: %q is not an integer: %w", string(s), err)
	}

	return value, nil
}

// IsZero reports whether the value is unset.
func (s StringOrInt) IsZero() bool {
	return s == ""
}

// UnmarshalJSON accepts a JSON string, a JSON number, or null.
func (s *StringOrInt) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*s = ""
		return nil
	}

	if trimmed[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*s = StringOrInt(str)
		return nil
	}

	// Bare number (or anything else JSON-valid). Keep the literal text so no
	// precision is lost.
	*s = StringOrInt(trimmed)
	return nil
}

// MarshalJSON emits a bare number when the value is an integer literal,
// otherwise a JSON string. Keeping non-integer values as strings means a value
// such as "0.00" does not silently change JSON type.
func (s StringOrInt) MarshalJSON() ([]byte, error) {
	if s == "" {
		return []byte(`""`), nil
	}

	if _, err := strconv.ParseInt(string(s), 10, 64); err == nil {
		return []byte(string(s)), nil
	}

	return json.Marshal(string(s))
}
