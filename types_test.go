package duitku

import (
	"encoding/json"
	"testing"
)

func TestStringOrIntUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    StringOrInt
		wantErr bool
	}{
		{name: "Quoted string", input: `"40000"`, want: "40000"},
		{name: "Bare number", input: `40000`, want: "40000"},
		{name: "Zero number", input: `0`, want: "0"},
		{name: "Empty string", input: `""`, want: ""},
		{name: "Null", input: `null`, want: ""},
		{name: "Quoted zero", input: `"0"`, want: "0"},
		{name: "Decimal string", input: `"0.00"`, want: "0.00"},
		{name: "Negative number", input: `-1500`, want: "-1500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got StringOrInt
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				if !tt.wantErr {
					t.Fatalf("UnmarshalJSON(%s) error = %v, want nil", tt.input, err)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("UnmarshalJSON(%s) error = nil, want error", tt.input)
			}
			if got != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringOrIntMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		value StringOrInt
		want  string
	}{
		{name: "Numeric marshals as a number", value: "40000", want: `40000`},
		{name: "Zero marshals as a number", value: "0", want: `0`},
		{name: "Empty marshals as an empty string", value: "", want: `""`},
		{name: "Non numeric marshals as a string", value: "0.00", want: `"0.00"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal(%q) error = %v, want nil", tt.value, err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal(%q) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

// TestStringOrIntRoundTrip proves a value survives a marshal/unmarshal cycle.
func TestStringOrIntRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "Quoted amount", input: `"40000"`},
		{name: "Bare amount", input: `40000`},
		{name: "Decimal fee", input: `"0.00"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var original StringOrInt
			if err := json.Unmarshal([]byte(tt.input), &original); err != nil {
				t.Fatalf("UnmarshalJSON(%s) error = %v", tt.input, err)
			}

			encoded, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal(%q) error = %v", original, err)
			}

			var decoded StringOrInt
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("UnmarshalJSON(%s) error = %v", encoded, err)
			}

			if decoded != original {
				t.Errorf("round trip = %q, want %q", decoded, original)
			}
		})
	}
}

func TestStringOrIntInt(t *testing.T) {
	tests := []struct {
		name    string
		value   StringOrInt
		want    int
		wantErr bool
	}{
		{name: "Numeric", value: "40000", want: 40000},
		{name: "Zero", value: "0", want: 0},
		{name: "Empty is zero", value: "", want: 0},
		{name: "Decimal fails", value: "0.00", wantErr: true},
		{name: "Text fails", value: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.Int()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Int() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("Int() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStringOrIntIsZeroAndString(t *testing.T) {
	tests := []struct {
		name       string
		value      StringOrInt
		wantIsZero bool
		wantString string
	}{
		{name: "Numeric", value: "40000", wantIsZero: false, wantString: "40000"},
		{name: "Empty", value: "", wantIsZero: true, wantString: ""},
		{name: "Zero", value: "0", wantIsZero: false, wantString: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.IsZero(); got != tt.wantIsZero {
				t.Errorf("IsZero() = %v, want %v", got, tt.wantIsZero)
			}
			if got := tt.value.String(); got != tt.wantString {
				t.Errorf("String() = %q, want %q", got, tt.wantString)
			}
		})
	}
}
