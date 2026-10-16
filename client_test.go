package duitku

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// testHMAC computes HMAC-SHA256 the way the Duitku docs specify, independently
// of the client implementation, so tests can cross-check a signature.
func testHMAC(apiKey, stringToSign string) string {
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil))
}

// legacyMD5Signature reproduces the obsolete scheme the SDK used to implement:
// a single MD5 over the message with the API key appended.
func legacyMD5Signature(stringToSign, apiKey string) string {
	sum := md5.Sum([]byte(stringToSign + apiKey))
	return hex.EncodeToString(sum[:])
}

// newTestClient builds a Client wired to an httptest server.
func newTestClient(server *httptest.Server) *Client {
	return &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
		logger:     log.New(io.Discard, "", 0),
	}
}

// unmarshalableBody cannot be marshaled to JSON.
var unmarshalableBody = make(chan int)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantBaseURL string
		wantTimeout time.Duration
	}{
		{
			name: "Sandbox",
			config: Config{
				MerchantCode: "DXXXX",
				APIKey:       "DXXXXCX80TZJ85Q70QCI",
				IsSandbox:    true,
			},
			wantBaseURL: SandboxBaseURL,
			wantTimeout: 30 * time.Second,
		},
		{
			name: "Production",
			config: Config{
				MerchantCode: "DXXXX",
				APIKey:       "DXXXXCX80TZJ85Q70QCI",
				IsSandbox:    false,
			},
			wantBaseURL: ProductionBaseURL,
			wantTimeout: 30 * time.Second,
		},
		{
			name: "Custom HTTP Client",
			config: Config{
				MerchantCode: "DXXXX",
				APIKey:       "DXXXXCX80TZJ85Q70QCI",
				IsSandbox:    true,
				HTTPClient:   &http.Client{Timeout: 60 * time.Second},
			},
			wantBaseURL: SandboxBaseURL,
			wantTimeout: 60 * time.Second,
		},
		{
			// A caller supplied client without a timeout must not be used
			// as-is, or every request would block forever.
			name: "Custom HTTP Client without a timeout gets the default",
			config: Config{
				MerchantCode: "DXXXX",
				APIKey:       "DXXXXCX80TZJ85Q70QCI",
				IsSandbox:    true,
				HTTPClient:   &http.Client{},
			},
			wantBaseURL: SandboxBaseURL,
			wantTimeout: 30 * time.Second,
		},
		{
			name: "Custom Logger",
			config: Config{
				MerchantCode: "DXXXX",
				APIKey:       "DXXXXCX80TZJ85Q70QCI",
				IsSandbox:    true,
				Logger:       log.New(os.Stderr, "test: ", log.LstdFlags),
			},
			wantBaseURL: SandboxBaseURL,
			wantTimeout: 30 * time.Second,
		},
		{
			name: "With Logging Enabled",
			config: Config{
				MerchantCode:               "DXXXX",
				APIKey:                     "DXXXXCX80TZJ85Q70QCI",
				IsSandbox:                  true,
				LogEveryRequestAndResponse: true,
			},
			wantBaseURL: SandboxBaseURL,
			wantTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)

			if client.baseURL != tt.wantBaseURL {
				t.Errorf("NewClient() baseURL = %v, want %v", client.baseURL, tt.wantBaseURL)
			}
			if client.config.MerchantCode != tt.config.MerchantCode {
				t.Errorf("NewClient() merchantCode = %v, want %v", client.config.MerchantCode, tt.config.MerchantCode)
			}
			if client.config.APIKey != tt.config.APIKey {
				t.Errorf("NewClient() apiKey = %v, want %v", client.config.APIKey, tt.config.APIKey)
			}
			if client.httpClient.Timeout != tt.wantTimeout {
				t.Errorf("NewClient() httpClient.Timeout = %v, want %v", client.httpClient.Timeout, tt.wantTimeout)
			}
			if client.logger == nil {
				t.Error("NewClient() logger is nil, want non-nil")
			}
			if client.logEveryRequestAndResponse != tt.config.LogEveryRequestAndResponse {
				t.Errorf("NewClient() logEveryRequestAndResponse = %v, want %v",
					client.logEveryRequestAndResponse, tt.config.LogEveryRequestAndResponse)
			}

			if tt.config.HTTPClient != nil && tt.config.HTTPClient.Timeout != 0 {
				if client.httpClient != tt.config.HTTPClient {
					t.Errorf("NewClient() httpClient = %v, want the caller supplied client", client.httpClient)
				}
			}
			if tt.config.Logger != nil && client.logger != tt.config.Logger {
				t.Errorf("NewClient() logger = %v, want the caller supplied logger", client.logger)
			}
		})
	}
}

// TestSignatureVectors pins the signature algorithm for every documented
// operation to HMAC_SHA256(stringToSign, apiKey) in lowercase hex.
//
// Each case asserts the exact digest against a fixed literal, and additionally
// checks the properties that the obsolete MD5 scheme violated: a 64 character
// length, lowercase output, and a value that differs from the legacy digest.
func TestSignatureVectors(t *testing.T) {
	const (
		apiKey = "DXXXXCX80TZJ85Q70QCI"
		// The API key is the HMAC key, not a suffix of the message.
		merchantCode = "DXXXX"
	)

	tests := []struct {
		name         string
		params       []string
		wantStringTo string
		expected     string
	}{
		{
			// merchantcode + amount + datetime
			name:         "Get Payment Method",
			params:       []string{merchantCode, "10000", "2022-01-25 16:23:08"},
			wantStringTo: "DXXXX100002022-01-25 16:23:08",
			expected:     "211cd5de6edf73a597ee6847e63d342346fbd0212604e6cbe544d97788817c82",
		},
		{
			// merchantCode + merchantOrderId + paymentAmount
			name:         "Create Transaction",
			params:       []string{merchantCode, "ORDER123", "40000"},
			wantStringTo: "DXXXXORDER12340000",
			expected:     "04c5dfd9467fd81ab3ee6def7e46d2af044bcf64208b6db998256d103e1b73ad",
		},
		{
			// merchantCode + merchantOrderId
			name:         "Check Transaction",
			params:       []string{merchantCode, "ORDER123"},
			wantStringTo: "DXXXXORDER123",
			expected:     "b80fd7de2ea8d95da86994ea0ba547bcdcd4451935929ee85ce52f04f3d98fd0",
		},
		{
			// merchantCode + amount + merchantOrderId
			name:         "Callback",
			params:       []string{merchantCode, "40000", "ORDER123"},
			wantStringTo: "DXXXX40000ORDER123",
			expected:     "48aed82d03914dc87da05226fd6c60b89f9a28ba81089e347e4c1de6552e2bfd",
		},
		{
			name:         "Empty params",
			params:       []string{},
			wantStringTo: "",
			expected:     "af23496fbb56b3d22f816f764d209a299df585d6c22e411ea205e4fb8bb61626",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(Config{
				MerchantCode: merchantCode,
				APIKey:       apiKey,
				IsSandbox:    true,
			})

			got := client.createSignatureHMACSHA256(tt.params...)

			if got != tt.expected {
				t.Errorf("createSignatureHMACSHA256() = %s, want %s", got, tt.expected)
			}

			// Cross-check against an independent HMAC implementation.
			if want := testHMAC(apiKey, tt.wantStringTo); got != want {
				t.Errorf("signature = %s, want %s from an independent HMAC", got, want)
			}

			if len(got) != 64 {
				t.Errorf("signature length = %d, want 64 (an MD5 digest would be 32)", len(got))
			}
			if strings.ToLower(got) != got {
				t.Errorf("signature = %s, want lowercase hex", got)
			}
			if got == legacyMD5Signature(tt.wantStringTo, apiKey) {
				t.Errorf("signature %s matches the obsolete MD5 scheme, want HMAC-SHA256", got)
			}
		})
	}
}

func TestDoRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		endpoint       string
		requestBody    interface{}
		nilResult      bool
		responseStatus int
		responseBody   string
		expectError    bool
		wantErrSubstr  string
	}{
		{
			name:           "Successful request",
			method:         http.MethodPost,
			endpoint:       "test-endpoint",
			requestBody:    map[string]string{"test": "data"},
			responseStatus: http.StatusOK,
			responseBody:   `{"responseCode":"00","responseMessage":"SUCCESS"}`,
		},
		{
			name:           "Successful request without a result target",
			method:         http.MethodPost,
			endpoint:       "test-endpoint",
			requestBody:    map[string]string{"test": "data"},
			nilResult:      true,
			responseStatus: http.StatusOK,
			responseBody:   `{"responseCode":"00","responseMessage":"SUCCESS"}`,
		},
		{
			name:           "Unmarshalable request body",
			method:         http.MethodPost,
			endpoint:       "test-endpoint",
			requestBody:    unmarshalableBody,
			responseStatus: http.StatusOK,
			responseBody:   `{}`,
			expectError:    true,
			wantErrSubstr:  "marshaling",
		},
		{
			name:           "Invalid JSON response",
			method:         http.MethodPost,
			endpoint:       "test-endpoint",
			responseStatus: http.StatusOK,
			responseBody:   `{invalid json}`,
			expectError:    true,
			wantErrSubstr:  "decoding",
		},
		{
			name:           "Error response",
			method:         http.MethodPost,
			endpoint:       "test-endpoint",
			responseStatus: http.StatusBadRequest,
			responseBody:   `{"responseCode":"01","responseMessage":"ERROR"}`,
			expectError:    true,
			wantErrSubstr:  "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("request method = %s, want %s", r.Method, tt.method)
				}
				if want := "/" + tt.endpoint; r.URL.Path != want {
					t.Errorf("request path = %s, want %s", r.URL.Path, want)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %s, want application/json", got)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server)

			var response struct {
				ResponseCode    string `json:"responseCode"`
				ResponseMessage string `json:"responseMessage"`
			}

			var result interface{} = &response
			if tt.nilResult {
				result = nil
			}

			err := client.doRequest(tt.method, tt.endpoint, tt.requestBody, result)

			if (err != nil) != tt.expectError {
				t.Fatalf("doRequest() error = %v, wantErr %v", err, tt.expectError)
			}
			if tt.expectError {
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("doRequest() error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}

			if tt.nilResult {
				return
			}
			if response.ResponseCode != "00" {
				t.Errorf("responseCode = %v, want 00", response.ResponseCode)
			}
			if response.ResponseMessage != "SUCCESS" {
				t.Errorf("responseMessage = %v, want SUCCESS", response.ResponseMessage)
			}
		})
	}
}

func TestDoRequestLogging(t *testing.T) {
	tests := []struct {
		name          string
		loggingOn     bool
		wantSubstring string
	}{
		{name: "Request method is logged", loggingOn: true, wantSubstring: "Request method: POST"},
		{name: "Request url is logged", loggingOn: true, wantSubstring: "Request url:"},
		{name: "Request body is logged", loggingOn: true, wantSubstring: "Request body:"},
		{name: "Response status code is logged", loggingOn: true, wantSubstring: "Response status code:"},
		{name: "Response body is logged", loggingOn: true, wantSubstring: "Response body:"},
		{name: "Nothing is logged when disabled", loggingOn: false, wantSubstring: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuffer := &bytes.Buffer{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"responseCode":"00","responseMessage":"SUCCESS"}`))
			}))
			defer server.Close()

			client := newTestClient(server)
			client.logger = log.New(logBuffer, "", 0)
			client.logEveryRequestAndResponse = tt.loggingOn

			var response struct {
				ResponseCode    string `json:"responseCode"`
				ResponseMessage string `json:"responseMessage"`
			}
			err := client.doRequest(http.MethodPost, "test-endpoint", map[string]string{"test": "data"}, &response)
			if err != nil {
				t.Fatalf("doRequest() error = %v, want nil", err)
			}

			got := logBuffer.String()

			if tt.wantSubstring == "" {
				if got != "" {
					t.Errorf("log output = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantSubstring) {
				t.Errorf("log output %q does not contain %q", got, tt.wantSubstring)
			}
		})
	}
}

func TestDoRequestRequiresCredentials(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantSubstr string
	}{
		{
			name:       "Empty MerchantCode",
			config:     Config{APIKey: "DXXXXCX80TZJ85Q70QCI"},
			wantSubstr: "MerchantCode is required",
		},
		{
			name:       "Empty APIKey",
			config:     Config{MerchantCode: "DXXXX"},
			wantSubstr: "APIKey is required",
		},
		{
			name:       "Both empty",
			config:     Config{},
			wantSubstr: "MerchantCode is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)

			var response struct{}
			err := client.doRequest(http.MethodPost, "test-endpoint", nil, &response)
			if err == nil {
				t.Fatal("doRequest() expected error for empty credentials, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("doRequest() error = %v, want it to contain %q", err, tt.wantSubstr)
			}
		})
	}
}

// TestAPIErrorParsing covers every error body shape Duitku is known to send.
// The capital-M "Message" key is the documented one for HTTP 400/401/404/409;
// the old code did not read it, so the reason was reported as empty.
func TestAPIErrorParsing(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		body        string
		wantMessage string
		wantCode    string
	}{
		{
			name:        "Documented Message key",
			statusCode:  http.StatusUnauthorized,
			body:        `{"Message":"Wrong signature"}`,
			wantMessage: "Wrong signature",
		},
		{
			name:        "Minimum payment message",
			statusCode:  http.StatusBadRequest,
			body:        `{"Message":"Minimum Payment 10000 IDR"}`,
			wantMessage: "Minimum Payment 10000 IDR",
		},
		{
			name:        "Payment channel message",
			statusCode:  http.StatusNotFound,
			body:        `{"Message":"Payment channel not available"}`,
			wantMessage: "Payment channel not available",
		},
		{
			name:        "Item total mismatch",
			statusCode:  http.StatusConflict,
			body:        `{"Message":"Payment amount must be equal to all item price"}`,
			wantMessage: "Payment amount must be equal to all item price",
		},
		{
			name:        "responseMessage fallback",
			statusCode:  http.StatusBadRequest,
			body:        `{"responseCode":"01","responseMessage":"ERROR"}`,
			wantMessage: "ERROR",
			wantCode:    "01",
		},
		{
			name:        "statusMessage fallback",
			statusCode:  http.StatusBadRequest,
			body:        `{"statusCode":"02","statusMessage":"CANCELLED"}`,
			wantMessage: "CANCELLED",
			wantCode:    "02",
		},
		{
			name:        "HTML gateway page is not a decoding failure",
			statusCode:  http.StatusBadGateway,
			body:        "<html><body>502 Bad Gateway</body></html>",
			wantMessage: "<html><body>502 Bad Gateway</body></html>",
		},
		{
			name:        "Empty body falls back to the status text",
			statusCode:  http.StatusInternalServerError,
			body:        "",
			wantMessage: http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(server)

			var response struct{}
			err := client.doRequest(http.MethodPost, "test-endpoint", nil, &response)
			if err == nil {
				t.Fatal("doRequest() expected error, got nil")
			}

			if strings.Contains(err.Error(), "decoding error response") {
				t.Errorf("error %q should not report a decoding failure", err.Error())
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("doRequest() error type = %T, want *APIError", err)
			}

			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("APIError.Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("APIError.Code = %q, want %q", apiErr.Code, tt.wantCode)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Errorf("error string %q does not contain %q", err.Error(), tt.wantMessage)
			}
		})
	}
}

func TestAPIErrorString(t *testing.T) {
	tests := []struct {
		name     string
		apiErr   *APIError
		expected string
	}{
		{
			name:     "With code",
			apiErr:   &APIError{StatusCode: http.StatusBadRequest, Code: "01", Message: "ERROR"},
			expected: "duitku: API error (HTTP 400, code 01): ERROR",
		},
		{
			name:     "Without code",
			apiErr:   &APIError{StatusCode: http.StatusUnauthorized, Message: "Wrong signature"},
			expected: "duitku: API error (HTTP 401): Wrong signature",
		},
		{
			name:     "Without message falls back to the status text",
			apiErr:   &APIError{StatusCode: http.StatusNotFound},
			expected: "duitku: API error (HTTP 404): Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apiErr.Error(); got != tt.expected {
				t.Errorf("APIError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{name: "Shorter than max", input: "abc", max: 10, expected: "abc"},
		{name: "Exactly max", input: "abcde", max: 5, expected: "abcde"},
		{name: "Longer than max", input: "abcdefgh", max: 5, expected: "abcde..."},
		{name: "Empty", input: "", max: 5, expected: ""},
		{name: "Multibyte counted by rune", input: "halo dunia", max: 4, expected: "halo..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.input, tt.max); got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expected)
			}
		})
	}
}
