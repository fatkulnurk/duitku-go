package duitku

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// expectedCallbackSignature is the documented vector for
// HMAC_SHA256("DXXXX" + "40000" + "ORDER123", "DXXXXCX80TZJ85Q70QCI").
const expectedCallbackSignature = "48aed82d03914dc87da05226fd6c60b89f9a28ba81089e347e4c1de6552e2bfd"

const (
	callbackMerchantCode = "DXXXX"
	callbackAPIKey       = "DXXXXCX80TZJ85Q70QCI"
)

// legacyCallbackSignature reproduces the obsolete MD5 scheme so tests can prove
// it is rejected.
func legacyCallbackSignature() string {
	sum := md5.Sum([]byte(callbackMerchantCode + "40000" + "ORDER123" + callbackAPIKey))
	return hex.EncodeToString(sum[:])
}

// newCallbackTestClient builds a client whose logger is discarded, keeping
// rejection logs out of the test output.
func newCallbackTestClient() *Client {
	return &Client{
		config: Config{
			MerchantCode: callbackMerchantCode,
			APIKey:       callbackAPIKey,
			IsSandbox:    true,
		},
		logger: log.New(io.Discard, "", 0),
	}
}

// validCallbackForm returns the documented callback payload.
func validCallbackForm() url.Values {
	form := url.Values{}
	form.Add("merchantCode", callbackMerchantCode)
	form.Add("amount", "40000")
	form.Add("merchantOrderId", "ORDER123")
	form.Add("productDetail", "Test Product")
	form.Add("additionalParam", "")
	form.Add("paymentCode", "VC")
	form.Add("resultCode", "00")
	form.Add("reference", "DXXXXCX80TZJ85Q70QCI")
	form.Add("signature", expectedCallbackSignature)

	return form
}

// newCallbackRequest builds a form-encoded callback request.
func newCallbackRequest(t *testing.T, form url.Values) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "/callback", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req
}

// TestParseCallback covers the documented callback fields and the request
// shapes a valid callback can arrive in.
func TestParseCallback(t *testing.T) {
	tests := []struct {
		name     string
		build    func(*url.Values)
		rawQuery string
		verify   func(*testing.T, *CallbackData)
	}{
		{
			name:  "Documented minimal callback",
			build: func(form *url.Values) {},
			verify: func(t *testing.T, data *CallbackData) {
				if data.MerchantCode != callbackMerchantCode {
					t.Errorf("MerchantCode = %s, want %s", data.MerchantCode, callbackMerchantCode)
				}
				if data.Amount != "40000" {
					t.Errorf("Amount = %s, want 40000", data.Amount)
				}
				if data.MerchantOrderID != "ORDER123" {
					t.Errorf("MerchantOrderID = %s, want ORDER123", data.MerchantOrderID)
				}
				if data.ProductDetail != "Test Product" {
					t.Errorf("ProductDetail = %s, want Test Product", data.ProductDetail)
				}
				if data.PaymentCode != "VC" {
					t.Errorf("PaymentCode = %s, want VC", data.PaymentCode)
				}
				if data.ResultCode != CallbackStatusSuccess {
					t.Errorf("ResultCode = %s, want 00", data.ResultCode)
				}
				if data.Reference != "DXXXXCX80TZJ85Q70QCI" {
					t.Errorf("Reference = %s, want DXXXXCX80TZJ85Q70QCI", data.Reference)
				}
				if data.Signature != expectedCallbackSignature {
					t.Errorf("Signature = %s, want %s", data.Signature, expectedCallbackSignature)
				}
				if !data.IsSuccessful() {
					t.Error("IsSuccessful() = false, want true")
				}
			},
		},
		{
			name: "All documented optional fields",
			build: func(form *url.Values) {
				form.Set("additionalParam", "contoh param")
				form.Set("merchantUserId", "user@example.com")
				form.Set("publisherOrderId", "MGUHWKJX3M1KMSQN5")
				form.Set("spUserHash", "xxxyyyzzz")
				form.Set("settlementDate", "2023-07-25")
				form.Set("issuerCode", "93600523")
				form.Set("customerName", "Bam**** Maul***")
			},
			verify: func(t *testing.T, data *CallbackData) {
				if data.AdditionalParam != "contoh param" {
					t.Errorf("AdditionalParam = %s, want contoh param", data.AdditionalParam)
				}
				if data.MerchantUserID != "user@example.com" {
					t.Errorf("MerchantUserID = %s, want user@example.com", data.MerchantUserID)
				}
				if data.PublisherOrderId != "MGUHWKJX3M1KMSQN5" {
					t.Errorf("PublisherOrderId = %s, want MGUHWKJX3M1KMSQN5", data.PublisherOrderId)
				}
				if data.SpUserHash != "xxxyyyzzz" {
					t.Errorf("SpUserHash = %s, want xxxyyyzzz", data.SpUserHash)
				}
				if data.SettlementDate != "2023-07-25" {
					t.Errorf("SettlementDate = %s, want 2023-07-25", data.SettlementDate)
				}
				if data.IssuerCode != "93600523" {
					t.Errorf("IssuerCode = %s, want 93600523", data.IssuerCode)
				}
				if data.CustomerName != "Bam**** Maul***" {
					t.Errorf("CustomerName = %s, want Bam**** Maul***", data.CustomerName)
				}
			},
		},
		{
			name:     "Query string cannot override signed fields",
			build:    func(form *url.Values) {},
			rawQuery: "merchantCode=DYYYY&amount=999999",
			verify: func(t *testing.T, data *CallbackData) {
				if data.MerchantCode != callbackMerchantCode {
					t.Errorf("MerchantCode = %s, want %s from the body", data.MerchantCode, callbackMerchantCode)
				}
				if data.Amount != "40000" {
					t.Errorf("Amount = %s, want 40000 from the body", data.Amount)
				}
			},
		},
		{
			name: "Failed resultCode is parsed and reported as failed",
			build: func(form *url.Values) {
				form.Set("resultCode", CallbackStatusFailed)
			},
			verify: func(t *testing.T, data *CallbackData) {
				if data.ResultCode != CallbackStatusFailed {
					t.Errorf("ResultCode = %s, want 01", data.ResultCode)
				}
				if data.IsSuccessful() {
					t.Error("IsSuccessful() = true, want false")
				}
				if !data.IsFailed() {
					t.Error("IsFailed() = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newCallbackTestClient()

			form := validCallbackForm()
			tt.build(&form)

			req := newCallbackRequest(t, form)
			req.URL.RawQuery = tt.rawQuery

			data, err := client.ParseCallback(req)
			if err != nil {
				t.Fatalf("ParseCallback() error = %v, want nil", err)
			}

			tt.verify(t, data)
		})
	}
}

// TestParseCallbackErrors covers every rejection path.
func TestParseCallbackErrors(t *testing.T) {
	tests := []struct {
		name          string
		build         func(*url.Values)
		rawBody       string
		wantErrSubstr string
	}{
		{
			name: "Missing merchantCode",
			build: func(form *url.Values) {
				form.Del("merchantCode")
			},
			wantErrSubstr: "missing required callback parameters",
		},
		{
			name: "Missing amount",
			build: func(form *url.Values) {
				form.Del("amount")
			},
			wantErrSubstr: "missing required callback parameters",
		},
		{
			name: "Missing merchantOrderId",
			build: func(form *url.Values) {
				form.Del("merchantOrderId")
			},
			wantErrSubstr: "missing required callback parameters",
		},
		{
			name: "Missing signature",
			build: func(form *url.Values) {
				form.Del("signature")
			},
			wantErrSubstr: "missing required callback parameters",
		},
		{
			name: "Empty form",
			build: func(form *url.Values) {
				*form = url.Values{}
			},
			wantErrSubstr: "missing required callback parameters",
		},
		{
			// resultCode is not part of the documented required set.
			name: "Missing resultCode is still accepted by validation",
			build: func(form *url.Values) {
				form.Del("resultCode")
			},
			wantErrSubstr: "",
		},
		{
			name: "Obsolete MD5 signature is rejected",
			build: func(form *url.Values) {
				form.Set("signature", legacyCallbackSignature())
			},
			wantErrSubstr: "invalid callback signature",
		},
		{
			name: "Garbage signature is rejected",
			build: func(form *url.Values) {
				form.Set("signature", "not-a-valid-signature")
			},
			wantErrSubstr: "invalid callback signature",
		},
		{
			name: "Uppercase signature is rejected",
			build: func(form *url.Values) {
				form.Set("signature", strings.ToUpper(expectedCallbackSignature))
			},
			wantErrSubstr: "invalid callback signature",
		},
		{
			name: "Truncated signature is rejected",
			build: func(form *url.Values) {
				form.Set("signature", expectedCallbackSignature[:32])
			},
			wantErrSubstr: "invalid callback signature",
		},
		{
			name: "Tampered amount breaks the signature",
			build: func(form *url.Values) {
				form.Set("amount", "999999")
			},
			wantErrSubstr: "invalid callback signature",
		},
		{
			name: "Tampered order id breaks the signature",
			build: func(form *url.Values) {
				form.Set("merchantOrderId", "ORDER999")
			},
			wantErrSubstr: "invalid callback signature",
		},
		{
			name:          "Malformed form body",
			rawBody:       "merchantCode=%zz",
			wantErrSubstr: "error parsing form",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newCallbackTestClient()

			var req *http.Request
			if tt.rawBody != "" {
				var err error
				req, err = http.NewRequest(http.MethodPost, "/callback", strings.NewReader(tt.rawBody))
				if err != nil {
					t.Fatalf("Error creating request: %v", err)
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				form := validCallbackForm()
				tt.build(&form)
				req = newCallbackRequest(t, form)
			}

			data, err := client.ParseCallback(req)

			if tt.wantErrSubstr == "" {
				if err != nil {
					t.Fatalf("ParseCallback() error = %v, want nil", err)
				}
				if data.ResultCode != "" {
					t.Errorf("ResultCode = %q, want empty", data.ResultCode)
				}
				return
			}

			if err == nil {
				t.Fatal("ParseCallback() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("ParseCallback() error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

// TestParseCallbackMerchantCodeMismatch covers the guard against applying
// another merchant's callback to this client. A callback signed correctly for a
// different merchant must still be rejected.
func TestParseCallbackMerchantCodeMismatch(t *testing.T) {
	tests := []struct {
		name         string
		merchantCode string
	}{
		{name: "Another merchant code", merchantCode: "DYYYY"},
		{name: "Another merchant code with a similar prefix", merchantCode: "DXXXX1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newCallbackTestClient()

			form := validCallbackForm()
			form.Set("merchantCode", tt.merchantCode)

			// Sign correctly for whatever merchant code is presented.
			form.Set("signature", testHMAC(callbackAPIKey, tt.merchantCode+"40000"+"ORDER123"))

			_, err := client.ParseCallback(newCallbackRequest(t, form))
			if err == nil {
				t.Fatal("ParseCallback() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), "merchant code") {
				t.Errorf("ParseCallback() error = %q, want it to mention the merchant code", err.Error())
			}
		})
	}
}

func TestVerifyCallbackSignature(t *testing.T) {
	tests := []struct {
		name string
		data *CallbackData
		want bool
	}{
		{
			name: "Valid HMAC signature",
			data: &CallbackData{
				MerchantCode:    callbackMerchantCode,
				Amount:          "40000",
				MerchantOrderID: "ORDER123",
				Signature:       expectedCallbackSignature,
			},
			want: true,
		},
		{
			name: "Obsolete MD5 signature",
			data: &CallbackData{
				MerchantCode:    callbackMerchantCode,
				Amount:          "40000",
				MerchantOrderID: "ORDER123",
				Signature:       legacyCallbackSignature(),
			},
			want: false,
		},
		{
			name: "Uppercase signature",
			data: &CallbackData{
				MerchantCode:    callbackMerchantCode,
				Amount:          "40000",
				MerchantOrderID: "ORDER123",
				Signature:       strings.ToUpper(expectedCallbackSignature),
			},
			want: false,
		},
		{
			name: "Tampered amount",
			data: &CallbackData{
				MerchantCode:    callbackMerchantCode,
				Amount:          "999999",
				MerchantOrderID: "ORDER123",
				Signature:       expectedCallbackSignature,
			},
			want: false,
		},
		{
			name: "Empty signature",
			data: &CallbackData{
				MerchantCode:    callbackMerchantCode,
				Amount:          "40000",
				MerchantOrderID: "ORDER123",
			},
			want: false,
		},
		{
			name: "Nil data",
			data: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newCallbackTestClient()

			if got := client.VerifyCallbackSignature(tt.data); got != tt.want {
				t.Errorf("VerifyCallbackSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCallbackStatusHelpers covers the callback resultCode semantics.
//
// The redirect codes reuse the same numbers with different meanings, so the
// only property that must hold everywhere is that a non-success code is never
// reported as successful.
func TestCallbackStatusHelpers(t *testing.T) {
	tests := []struct {
		name           string
		resultCode     string
		wantSuccessful bool
		wantFailed     bool
	}{
		{name: "Callback success", resultCode: CallbackStatusSuccess, wantSuccessful: true},
		{name: "Callback failed", resultCode: CallbackStatusFailed, wantFailed: true},
		{
			// "01" means failed on a callback and pending on a redirect. The
			// values collide, so IsFailed reports true; the safety property
			// that matters is that IsSuccessful stays false.
			name:       "Redirect pending shares the failed code",
			resultCode: RedirectStatusPending,
			wantFailed: true,
		},
		{name: "Redirect canceled is not a success", resultCode: RedirectStatusCanceled},
		{name: "Unknown code", resultCode: "99"},
		{name: "Empty code", resultCode: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &CallbackData{ResultCode: tt.resultCode}

			if got := data.IsSuccessful(); got != tt.wantSuccessful {
				t.Errorf("IsSuccessful() = %v, want %v", got, tt.wantSuccessful)
			}
			if got := data.IsFailed(); got != tt.wantFailed {
				t.Errorf("IsFailed() = %v, want %v", got, tt.wantFailed)
			}
		})
	}
}

// TestHandleCallback covers the full request path: status codes, whether the
// handler ran, and what gets logged.
func TestHandleCallback(t *testing.T) {
	tests := []struct {
		name              string
		method            string
		rawBody           string
		build             func(*url.Values)
		handlerErr        error
		wantStatus        int
		wantBody          string
		wantHandlerCalled bool
		wantLogContains   string
		wantLogAbsent     string
	}{
		{
			name:              "Valid callback succeeds",
			method:            http.MethodPost,
			wantStatus:        http.StatusOK,
			wantBody:          "OK",
			wantHandlerCalled: true,
		},
		{
			name:   "Obsolete MD5 signature is rejected",
			method: http.MethodPost,
			build: func(form *url.Values) {
				form.Set("signature", legacyCallbackSignature())
			},
			wantStatus:      http.StatusBadRequest,
			wantLogContains: "rejecting callback",
		},
		{
			name:   "Garbage signature is rejected",
			method: http.MethodPost,
			build: func(form *url.Values) {
				form.Set("signature", "wrong")
			},
			wantStatus:      http.StatusBadRequest,
			wantLogContains: "rejecting callback",
		},
		{
			name:   "Missing required parameters",
			method: http.MethodPost,
			build: func(form *url.Values) {
				*form = url.Values{"invalid": []string{"data"}}
			},
			wantStatus:      http.StatusBadRequest,
			wantLogContains: "rejecting callback",
		},
		{
			name:            "Malformed form body",
			method:          http.MethodPost,
			rawBody:         "merchantCode=%zz",
			wantStatus:      http.StatusBadRequest,
			wantLogContains: "rejecting callback",
		},
		{
			name:       "GET is not allowed",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   http.StatusText(http.StatusMethodNotAllowed) + "\n",
		},
		{
			// A handler failure answers 500 so Duitku retries, and the
			// internal error is logged rather than returned to the caller.
			name:              "Handler error returns 500 and hides the reason",
			method:            http.MethodPost,
			handlerErr:        fmt.Errorf("database exploded: connection refused"),
			wantStatus:        http.StatusInternalServerError,
			wantBody:          http.StatusText(http.StatusInternalServerError) + "\n",
			wantHandlerCalled: true,
			wantLogContains:   "database exploded",
			wantLogAbsent:     "database exploded: connection refused", // not in the response
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuffer := &bytes.Buffer{}
			client := newCallbackTestClient()
			client.logger = log.New(logBuffer, "", 0)

			var req *http.Request
			if tt.rawBody != "" {
				var err error
				req, err = http.NewRequest(tt.method, "/callback", strings.NewReader(tt.rawBody))
				if err != nil {
					t.Fatalf("Error creating request: %v", err)
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else if tt.method == http.MethodGet {
				var err error
				req, err = http.NewRequest(tt.method, "/callback", nil)
				if err != nil {
					t.Fatalf("Error creating request: %v", err)
				}
			} else {
				form := validCallbackForm()
				if tt.build != nil {
					tt.build(&form)
				}
				req = newCallbackRequest(t, form)
			}

			rr := httptest.NewRecorder()

			var handlerCalled bool
			var received *CallbackData
			handler := func(data *CallbackData) error {
				handlerCalled = true
				received = data
				return tt.handlerErr
			}

			client.HandleCallback(rr, req, handler)

			if handlerCalled != tt.wantHandlerCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantHandlerCalled)
			}
			if rr.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && rr.Body.String() != tt.wantBody {
				t.Errorf("response body = %q, want %q", rr.Body.String(), tt.wantBody)
			}
			if tt.wantHandlerCalled && received != nil && received.MerchantOrderID != "ORDER123" {
				t.Errorf("handler received order %s, want ORDER123", received.MerchantOrderID)
			}

			logOutput := logBuffer.String()
			if tt.wantLogContains != "" && !strings.Contains(logOutput, tt.wantLogContains) {
				t.Errorf("log %q does not contain %q", logOutput, tt.wantLogContains)
			}
			if tt.wantLogAbsent != "" && strings.Contains(rr.Body.String(), tt.wantLogAbsent) {
				t.Errorf("response body leaked the internal error: %q", rr.Body.String())
			}
			if tt.name == "Valid callback succeeds" && logOutput != "" {
				t.Errorf("log = %q, want empty for a successful callback", logOutput)
			}
		})
	}
}
