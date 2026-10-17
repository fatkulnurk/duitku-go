package duitku

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// datetimePattern is the documented format for the get payment method datetime
// parameter: yyyy-MM-dd HH:mm:ss.
var datetimePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)

// getPaymentMethodsRequestBody mirrors the documented request with explicit
// tags, including the lowercase merchantcode key this endpoint requires.
type getPaymentMethodsRequestBody struct {
	MerchantCode string `json:"merchantcode"`
	Amount       int    `json:"amount"`
	DateTime     string `json:"datetime"`
	Signature    string `json:"signature"`
}

// capturePaymentMethodServer starts a server that records the request and
// replies with the given body.
func capturePaymentMethodServer(t *testing.T, status int, responseBody string) (*httptest.Server, *getPaymentMethodsRequestBody) {
	t.Helper()

	captured := &getPaymentMethodsRequestBody{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("request method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", got)
		}

		if err := json.NewDecoder(r.Body).Decode(captured); err != nil {
			t.Errorf("decoding request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	return server, captured
}

// TestGetPaymentMethods covers parsing of the documented payment fee list.
func TestGetPaymentMethods(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		want         []PaymentMethod
	}{
		{
			name: "Documented example with two methods",
			responseBody: `{
				"paymentFee": [
					{
						"paymentMethod": "VA",
						"paymentName": "MAYBANK VA",
						"paymentImage": "https://images.duitku.com/hotlink-ok/VA.PNG",
						"totalFee": "0"
					},
					{
						"paymentMethod": "BT",
						"paymentName": "PERMATA VA",
						"paymentImage": "https://images.duitku.com/hotlink-ok/PERMATA.PNG",
						"totalFee": "0"
					}
				],
				"responseCode": "00",
				"responseMessage": "SUCCESS"
			}`,
			want: []PaymentMethod{
				{
					PaymentMethod: "VA",
					PaymentName:   "MAYBANK VA",
					PaymentImage:  "https://images.duitku.com/hotlink-ok/VA.PNG",
					TotalFee:      "0",
				},
				{
					PaymentMethod: "BT",
					PaymentName:   "PERMATA VA",
					PaymentImage:  "https://images.duitku.com/hotlink-ok/PERMATA.PNG",
					TotalFee:      "0",
				},
			},
		},
		{
			name: "Fee charged to the customer",
			responseBody: `{
				"paymentFee": [
					{
						"paymentMethod": "OV",
						"paymentName": "OVO",
						"paymentImage": "https://images.duitku.com/hotlink-ok/OVO.PNG",
						"totalFee": "1500"
					}
				],
				"responseCode": "00",
				"responseMessage": "SUCCESS"
			}`,
			want: []PaymentMethod{
				{
					PaymentMethod: "OV",
					PaymentName:   "OVO",
					PaymentImage:  "https://images.duitku.com/hotlink-ok/OVO.PNG",
					TotalFee:      "1500",
				},
			},
		},
		{
			name:         "No active methods",
			responseBody: `{"paymentFee":[],"responseCode":"00","responseMessage":"SUCCESS"}`,
			want:         []PaymentMethod{},
		},
		{
			name:         "Missing paymentFee key",
			responseBody: `{"responseCode":"00","responseMessage":"SUCCESS"}`,
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, captured := capturePaymentMethodServer(t, http.StatusOK, tt.responseBody)

			client := newTestClient(server)

			methods, err := client.GetPaymentMethods(10000)
			if err != nil {
				t.Fatalf("GetPaymentMethods() error = %v, want nil", err)
			}

			if captured.MerchantCode != "DXXXX" {
				t.Errorf("merchantcode = %s, want DXXXX", captured.MerchantCode)
			}
			if captured.Amount != 10000 {
				t.Errorf("amount = %d, want 10000", captured.Amount)
			}
			if !datetimePattern.MatchString(captured.DateTime) {
				t.Errorf("datetime = %q, want the format yyyy-MM-dd HH:mm:ss", captured.DateTime)
			}
			if len(captured.Signature) != 64 {
				t.Errorf("signature length = %d, want 64 (an MD5 digest would be 32)", len(captured.Signature))
			}

			if len(methods) != len(tt.want) {
				t.Fatalf("GetPaymentMethods() returned %d methods, want %d", len(methods), len(tt.want))
			}
			for i, want := range tt.want {
				if methods[i] != want {
					t.Errorf("methods[%d] = %+v, want %+v", i, methods[i], want)
				}
			}
		})
	}
}

// TestGetPaymentMethodsSignature pins the get payment method signature to
// HMAC_SHA256(merchantcode + amount + datetime, apiKey).
func TestGetPaymentMethodsSignature(t *testing.T) {
	const apiKey = "DXXXXCX80TZJ85Q70QCI"

	tests := []struct {
		name     string
		amount   string
		datetime string
		expected string
	}{
		{
			name:     "Documented vector",
			amount:   "10000",
			datetime: "2022-01-25 16:23:08",
			expected: "211cd5de6edf73a597ee6847e63d342346fbd0212604e6cbe544d97788817c82",
		},
		{
			name:     "Larger amount",
			amount:   "150000",
			datetime: "2026-10-17 06:00:00",
			expected: testHMAC(apiKey, "DXXXX1500002026-10-17 06:00:00"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(Config{
				MerchantCode: "DXXXX",
				APIKey:       apiKey,
				IsSandbox:    true,
			})

			got := client.createSignatureHMACSHA256("DXXXX", tt.amount, tt.datetime)

			if got != tt.expected {
				t.Errorf("signature = %s, want %s", got, tt.expected)
			}
			if len(tt.expected) != 64 {
				t.Errorf("expected signature length = %d, want 64", len(tt.expected))
			}
			if tt.expected == legacyMD5Signature("DXXXX"+tt.amount+tt.datetime, apiKey) {
				t.Error("expected signature matches the obsolete MD5 scheme, want HMAC-SHA256")
			}
		})
	}
}

// failingRoundTripper simulates a transport failure without touching the
// network, so the test cannot flake on DNS.
type failingRoundTripper struct {
	err error
}

func (f failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.err
}

// TestGetPaymentMethodsErrors covers every failure mode of the endpoint.
func TestGetPaymentMethodsErrors(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		responseBody  string
		transportErr  error
		wantErrSubstr string
	}{
		{
			name:          "Failed responseCode in a 200 response",
			status:        http.StatusOK,
			responseBody:  `{"paymentFee":[],"responseCode":"01","responseMessage":"ERROR"}`,
			wantErrSubstr: "error getting payment methods: ERROR (responseCode 01)",
		},
		{
			name:          "HTTP 500 with the documented Message key",
			status:        http.StatusInternalServerError,
			responseBody:  `{"Message":"Internal Server Error"}`,
			wantErrSubstr: "duitku: API error (HTTP 500): Internal Server Error",
		},
		{
			name:          "Merchant not found",
			status:        http.StatusNotFound,
			responseBody:  `{"Message":"Merchant not found"}`,
			wantErrSubstr: "duitku: API error (HTTP 404): Merchant not found",
		},
		{
			name:          "Gateway error page",
			status:        http.StatusBadGateway,
			responseBody:  "<html><body>502 Bad Gateway</body></html>",
			wantErrSubstr: "duitku: API error (HTTP 502)",
		},
		{
			name:          "Invalid JSON body",
			status:        http.StatusOK,
			responseBody:  `{invalid json}`,
			wantErrSubstr: "decoding",
		},
		{
			name:          "Connection refused",
			transportErr:  &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			wantErrSubstr: "making request",
		},
		{
			name:          "DNS failure",
			transportErr:  &net.DNSError{Err: "no such host", Name: "sandbox.duitku.com"},
			wantErrSubstr: "making request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client

			if tt.transportErr != nil {
				client = &Client{
					config: Config{
						MerchantCode: "DXXXX",
						APIKey:       "DXXXXCX80TZJ85Q70QCI",
					},
					baseURL:    "https://sandbox.duitku.com/webapi/api",
					httpClient: &http.Client{Transport: failingRoundTripper{err: tt.transportErr}},
					logger:     log.New(io.Discard, "", 0),
				}
			} else {
				server, _ := capturePaymentMethodServer(t, tt.status, tt.responseBody)
				client = newTestClient(server)
			}

			_, err := client.GetPaymentMethods(10000)
			if err == nil {
				t.Fatal("GetPaymentMethods() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("GetPaymentMethods() error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

// TestGetPaymentMethodsEndpoint covers the URL path the method targets.
func TestGetPaymentMethodsEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "Posts to merchant/paymentmethod/getpaymentmethod",
			endpoint: "/merchant/paymentmethod/getpaymentmethod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"paymentFee":[],"responseCode":"00","responseMessage":"SUCCESS"}`))
			}))
			defer server.Close()

			client := newTestClient(server)

			if _, err := client.GetPaymentMethods(10000); err != nil {
				t.Fatalf("GetPaymentMethods() error = %v, want nil", err)
			}
			if gotPath != tt.endpoint {
				t.Errorf("request path = %s, want %s", gotPath, tt.endpoint)
			}
		})
	}
}
