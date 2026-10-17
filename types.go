package duitku

import (
	"encoding/json/jsontext"
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
//
// It implements the encoding/json/v2 MarshalerTo and UnmarshalerFrom
// interfaces. In Go 1.27 the encoding/json package is implemented on top of
// encoding/json/v2, so those two methods also serve callers that still import
// encoding/json.
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

// isIntegerLiteral reports whether the value should be sent as a bare JSON
// number rather than a quoted string. Keeping non-integer values as strings
// means a value such as "0.00" does not silently change JSON type.
func (s StringOrInt) isIntegerLiteral() bool {
	if s == "" {
		return false
	}

	_, err := strconv.ParseInt(string(s), 10, 64)

	return err == nil
}

// setFromJSONValue stores a decoded JSON value: a quoted string is unescaped, a
// bare number keeps its literal text so no precision is lost, and null clears
// the value.
func (s *StringOrInt) setFromJSONValue(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*s = ""
		return nil
	}

	if trimmed[0] != '"' {
		*s = StringOrInt(trimmed)
		return nil
	}

	// Reading a token rather than slicing off the quotes so that escape
	// sequences are decoded properly.
	decoder := jsontext.NewDecoder(strings.NewReader(trimmed))
	token, err := decoder.ReadToken()
	if err != nil {
		return fmt.Errorf("duitku: %s is not a valid JSON string: %w", trimmed, err)
	}

	*s = StringOrInt(token.String())

	return nil
}

// MarshalJSONTo implements the encoding/json/v2 MarshalerTo interface.
func (s StringOrInt) MarshalJSONTo(enc *jsontext.Encoder) error {
	if s.isIntegerLiteral() {
		return enc.WriteValue(jsontext.Value(s))
	}

	return enc.WriteToken(jsontext.String(string(s)))
}

// UnmarshalJSONFrom implements the encoding/json/v2 UnmarshalerFrom interface.
func (s *StringOrInt) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	value, err := dec.ReadValue()
	if err != nil {
		return err
	}

	return s.setFromJSONValue(value)
}
