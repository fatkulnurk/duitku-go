package duitku

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"strings"
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

// TestStringOrIntUnderJSONv2 exercises the type through encoding/json/v2, which
// is the package the client actually uses. It covers the v2 native
// MarshalJSONTo and UnmarshalJSONFrom methods.
func TestStringOrIntUnderJSONv2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     StringOrInt
		wantJSON string
	}{
		{
			name:     "Quoted amount",
			input:    `{"amount":"40000"}`,
			want:     "40000",
			wantJSON: `{"amount":40000}`,
		},
		{
			name:     "Bare amount",
			input:    `{"amount":40000}`,
			want:     "40000",
			wantJSON: `{"amount":40000}`,
		},
		{
			name:     "Decimal fee stays a string",
			input:    `{"amount":"0.00"}`,
			want:     "0.00",
			wantJSON: `{"amount":"0.00"}`,
		},
		{
			name:     "Null clears the value",
			input:    `{"amount":null}`,
			want:     "",
			wantJSON: `{"amount":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var holder struct {
				Amount StringOrInt `json:"amount"`
			}

			if err := jsonv2.Unmarshal([]byte(tt.input), &holder); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v, want nil", tt.input, err)
			}
			if holder.Amount != tt.want {
				t.Errorf("Unmarshal(%s) = %q, want %q", tt.input, holder.Amount, tt.want)
			}

			encoded, err := jsonv2.Marshal(holder)
			if err != nil {
				t.Fatalf("Marshal() error = %v, want nil", err)
			}
			if string(encoded) != tt.wantJSON {
				t.Errorf("Marshal() = %s, want %s", encoded, tt.wantJSON)
			}
		})
	}
}

// TestStringOrIntInterfacesAgree proves the encoding/json and encoding/json/v2
// code paths produce identical bytes, so a caller can use either package.
func TestStringOrIntInterfacesAgree(t *testing.T) {
	tests := []struct {
		name  string
		value StringOrInt
	}{
		{name: "Numeric", value: "40000"},
		{name: "Zero", value: "0"},
		{name: "Empty", value: ""},
		{name: "Decimal", value: "0.00"},
		{name: "Non numeric", value: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			legacy, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("encoding/json Marshal(%q) error = %v, want nil", tt.value, err)
			}

			native, err := jsonv2.Marshal(tt.value)
			if err != nil {
				t.Fatalf("encoding/json/v2 Marshal(%q) error = %v, want nil", tt.value, err)
			}

			if string(legacy) != string(native) {
				t.Errorf("encoding/json produced %s but encoding/json/v2 produced %s", legacy, native)
			}
		})
	}
}

// TestStringOrIntRejectsMalformedInput covers the error paths. The encoding
// interfaces only ever hand over validated JSON, so malformed input is fed in
// directly to exercise the guards.
func TestStringOrIntRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "Unterminated string", input: `"40000`},
		{name: "Lone quote", input: `"`},
		{name: "Invalid escape sequence", input: `"a\qb"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var direct StringOrInt
			if err := direct.setFromJSONValue([]byte(tt.input)); err == nil {
				t.Errorf("setFromJSONValue(%s) error = nil, want an error", tt.input)
			}

			decoder := jsontext.NewDecoder(strings.NewReader(tt.input))

			var fromDecoder StringOrInt
			if err := fromDecoder.UnmarshalJSONFrom(decoder); err == nil {
				t.Errorf("UnmarshalJSONFrom(%s) error = nil, want an error", tt.input)
			}
		})
	}
}

// TestStringOrIntUnescapesStrings proves a quoted value is decoded rather than
// taken literally, which matters because the raw text is stored in the type.
func TestStringOrIntUnescapesStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  StringOrInt
	}{
		{name: "Plain digits", input: `"40000"`, want: "40000"},
		{name: "Escaped quote", input: `"a\"b"`, want: `a"b`},
		{name: "Escaped newline", input: `"a\nb"`, want: "a\nb"},
		{name: "Unicode escape", input: `"\u0034\u0030"`, want: "40"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value StringOrInt
			if err := jsonv2.Unmarshal([]byte(tt.input), &value); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v, want nil", tt.input, err)
			}
			if value != tt.want {
				t.Errorf("Unmarshal(%s) = %q, want %q", tt.input, value, tt.want)
			}
		})
	}
}
