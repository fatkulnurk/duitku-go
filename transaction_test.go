package duitku

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// Fixed signature vectors. The API key is DXXXXCX80TZJ85Q70QCI and the
// merchant code is DXXXX.
const (
	expectedInquirySignature = "04c5dfd9467fd81ab3ee6def7e46d2af044bcf64208b6db998256d103e1b73ad"
	expectedCheckSignature   = "b80fd7de2ea8d95da86994ea0ba547bcdcd4451935929ee85ce52f04f3d98fd0"
)

// captureServer starts a test server that records the decoded request body and
// replies with the given status and body.
func captureServer(t *testing.T, status int, responseBody string) (*httptest.Server, *map[string]json.RawMessage) {
	t.Helper()

	captured := &map[string]json.RawMessage{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("request method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", got)
		}

		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		*captured = body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	return server, captured
}

// baseTransactionRequest returns a valid inquiry request used as the baseline
// for the tables below.
func baseTransactionRequest() TransactionRequest {
	return TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   PaymentMethodCreditCard,
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		CustomerVaName:  "John Doe",
		Email:           "customer@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    10,
	}
}

func assertJSONString(t *testing.T, body map[string]json.RawMessage, key, want string) {
	t.Helper()

	raw, ok := body[key]
	if !ok {
		t.Errorf("request body is missing key %q; body = %v", key, body)
		return
	}

	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Errorf("key %q is not a string: %s", key, raw)
		return
	}
	if got != want {
		t.Errorf("body[%q] = %q, want %q", key, got, want)
	}
}

func assertJSONNumber(t *testing.T, body map[string]json.RawMessage, key string, want int) {
	t.Helper()

	raw, ok := body[key]
	if !ok {
		t.Errorf("request body is missing key %q; body = %v", key, body)
		return
	}

	var got int
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Errorf("key %q is not a number: %s", key, raw)
		return
	}
	if got != want {
		t.Errorf("body[%q] = %d, want %d", key, got, want)
	}
}

func assertJSONAbsent(t *testing.T, body map[string]json.RawMessage, key string) {
	t.Helper()

	if _, ok := body[key]; ok {
		t.Errorf("request body contains %q, want it omitted; body = %v", key, body)
	}
}

func assertJSONObject(t *testing.T, body map[string]json.RawMessage, key string) map[string]json.RawMessage {
	t.Helper()

	raw, ok := body[key]
	if !ok {
		t.Fatalf("request body is missing object %q; body = %v", key, body)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("key %q is not an object: %s", key, raw)
	}

	return obj
}

func assertJSONArray(t *testing.T, body map[string]json.RawMessage, key string) []json.RawMessage {
	t.Helper()

	raw, ok := body[key]
	if !ok {
		t.Fatalf("request body is missing array %q; body = %v", key, body)
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		t.Fatalf("key %q is not an array: %s", key, raw)
	}

	return arr
}

// TestCreateTransaction covers the inquiry request and response across the
// documented channel shapes and response encodings.
func TestCreateTransaction(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(*TransactionRequest)
		responseBody string
		verify       func(*testing.T, *TransactionResponse)
	}{
		{
			name:   "Virtual account",
			mutate: func(r *TransactionRequest) { r.PaymentMethod = PaymentMethodBCA },
			responseBody: `{
				"merchantCode": "DXXXX",
				"reference": "DXXXXCX80TZJ85Q70QCI",
				"paymentUrl": "https://sandbox.duitku.com/topup/topupdirectv2.aspx?ref=BCA7WZ7EIDXXXXWEC",
				"vaNumber": "7007014001444348",
				"amount": "40000",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			verify: func(t *testing.T, response *TransactionResponse) {
				if response.VANumber != "7007014001444348" {
					t.Errorf("VANumber = %s, want 7007014001444348", response.VANumber)
				}
				if response.Reference != "DXXXXCX80TZJ85Q70QCI" {
					t.Errorf("Reference = %s, want DXXXXCX80TZJ85Q70QCI", response.Reference)
				}
				if response.PaymentURL == "" {
					t.Error("PaymentURL is empty, want the payment page URL")
				}
			},
		},
		{
			name:   "QRIS returns a qrString",
			mutate: func(r *TransactionRequest) { r.PaymentMethod = PaymentMethodQrisNobu },
			responseBody: `{
				"merchantCode": "DXXXX",
				"reference": "QRIS123456789",
				"qrString": "00020101021226660014ID.DANA.WWW0118936009110021515001",
				"amount": "40000",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			verify: func(t *testing.T, response *TransactionResponse) {
				if response.QrString != "00020101021226660014ID.DANA.WWW0118936009110021515001" {
					t.Errorf("QrString = %s, want the documented QR payload", response.QrString)
				}
				if response.VANumber != "" {
					t.Errorf("VANumber = %s, want empty for QRIS", response.VANumber)
				}
			},
		},
		{
			// The docs example spells the key "AppUrl" while the parameter
			// table spells it "appUrl".
			name:   "E-commerce AppUrl casing from the JSON example",
			mutate: func(r *TransactionRequest) { r.PaymentMethod = PaymentMethodTokopediaCard },
			responseBody: `{
				"merchantCode": "DXXXX",
				"reference": "T1ORDER123",
				"AppUrl": "https://tokopedia.app.link/?desktop_url=abc",
				"amount": "40000",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			verify: func(t *testing.T, response *TransactionResponse) {
				if response.AppURL != "https://tokopedia.app.link/?desktop_url=abc" {
					t.Errorf("AppURL = %s, want the tokopedia app link", response.AppURL)
				}
			},
		},
		{
			name:   "E-commerce appUrl casing from the parameter table",
			mutate: func(r *TransactionRequest) { r.PaymentMethod = PaymentMethodTokopediaEWallet },
			responseBody: `{
				"merchantCode": "DXXXX",
				"reference": "T2ORDER123",
				"appUrl": "https://tokopedia.app.link/?desktop_url=xyz",
				"amount": "40000",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			verify: func(t *testing.T, response *TransactionResponse) {
				if response.AppURL != "https://tokopedia.app.link/?desktop_url=xyz" {
					t.Errorf("AppURL = %s, want the tokopedia app link", response.AppURL)
				}
			},
		},
		{
			name:   "Amount quoted as a string",
			mutate: func(r *TransactionRequest) { r.PaymentMethod = PaymentMethodBCA },
			responseBody: `{
				"merchantCode": "DXXXX",
				"reference": "REF",
				"amount": "40000",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			verify: func(t *testing.T, response *TransactionResponse) {
				if response.Amount != "40000" {
					t.Errorf("Amount = %q, want %q", response.Amount, "40000")
				}
				if got, err := response.Amount.Int(); err != nil || got != 40000 {
					t.Errorf("Amount.Int() = (%d, %v), want (40000, nil)", got, err)
				}
			},
		},
		{
			// The response table types amount as an integer.
			name:   "Amount sent as a bare number",
			mutate: func(r *TransactionRequest) { r.PaymentMethod = PaymentMethodBCA },
			responseBody: `{
				"merchantCode": "DXXXX",
				"reference": "REF",
				"amount": 40000,
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			verify: func(t *testing.T, response *TransactionResponse) {
				if response.Amount != "40000" {
					t.Errorf("Amount = %q, want %q", response.Amount, "40000")
				}
			},
		},
		{
			name:   "Amount omitted",
			mutate: func(r *TransactionRequest) { r.PaymentMethod = PaymentMethodBCA },
			responseBody: `{
				"merchantCode": "DXXXX",
				"reference": "REF",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			verify: func(t *testing.T, response *TransactionResponse) {
				if !response.Amount.IsZero() {
					t.Errorf("Amount = %q, want it to be unset", response.Amount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, captured := captureServer(t, http.StatusOK, tt.responseBody)

			client := newTestClient(server)

			request := baseTransactionRequest()
			tt.mutate(&request)

			response, err := client.CreateTransaction(request)
			if err != nil {
				t.Fatalf("CreateTransaction() error = %v, want nil", err)
			}

			body := *captured

			assertJSONString(t, body, "merchantCode", "DXXXX")
			assertJSONNumber(t, body, "paymentAmount", request.PaymentAmount)
			assertJSONString(t, body, "paymentMethod", request.PaymentMethod)
			assertJSONString(t, body, "merchantOrderId", request.MerchantOrderID)
			assertJSONString(t, body, "signature", expectedInquirySignature)

			if response.MerchantCode != "DXXXX" {
				t.Errorf("Response MerchantCode = %s, want DXXXX", response.MerchantCode)
			}
			if response.StatusCode != StatusSuccess {
				t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
			}
			if response.StatusMessage != "SUCCESS" {
				t.Errorf("Response StatusMessage = %s, want SUCCESS", response.StatusMessage)
			}

			tt.verify(t, response)
		})
	}
}

// TestCreateTransactionRequestSerialization covers which keys reach the wire for
// each optional field.
func TestCreateTransactionRequestSerialization(t *testing.T) {
	const successBody = `{"merchantCode":"DXXXX","reference":"REF","amount":"40000","statusCode":"00","statusMessage":"SUCCESS"}`

	tests := []struct {
		name   string
		mutate func(*TransactionRequest)
		assert func(*testing.T, map[string]json.RawMessage)
	}{
		{
			name:   "Required fields are always present",
			mutate: func(r *TransactionRequest) {},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				assertJSONString(t, body, "merchantCode", "DXXXX")
				assertJSONNumber(t, body, "paymentAmount", 40000)
				assertJSONString(t, body, "merchantOrderId", "ORDER123")
				assertJSONString(t, body, "productDetails", "Test Product")
				assertJSONString(t, body, "email", "customer@example.com")
				assertJSONString(t, body, "paymentMethod", PaymentMethodCreditCard)
				assertJSONString(t, body, "customerVaName", "John Doe")
				assertJSONString(t, body, "returnUrl", "https://example.com/return")
				assertJSONString(t, body, "callbackUrl", "https://example.com/callback")
				assertJSONString(t, body, "signature", expectedInquirySignature)
			},
		},
		{
			name:   "Empty optional scalars are omitted",
			mutate: func(r *TransactionRequest) {},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				for _, key := range []string{"additionalParam", "merchantUserInfo", "phoneNumber", "itemDetails", "customerDetail", "accountLink", "creditCardDetail"} {
					assertJSONAbsent(t, body, key)
				}
			},
		},
		{
			name: "Populated optional scalars are sent",
			mutate: func(r *TransactionRequest) {
				r.AdditionalParam = "contoh param"
				r.MerchantUserInfo = "user@example.com"
				r.PhoneNumber = "08123456789"
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				assertJSONString(t, body, "additionalParam", "contoh param")
				assertJSONString(t, body, "merchantUserInfo", "user@example.com")
				assertJSONString(t, body, "phoneNumber", "08123456789")
			},
		},
		{
			name:   "Unset expiryPeriod is omitted",
			mutate: func(r *TransactionRequest) { r.ExpiryPeriod = 0 },
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				assertJSONAbsent(t, body, "expiryPeriod")
			},
		},
		{
			name:   "Set expiryPeriod is sent",
			mutate: func(r *TransactionRequest) { r.ExpiryPeriod = 1440 },
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				assertJSONNumber(t, body, "expiryPeriod", 1440)
			},
		},
		{
			name: "customerDetail omits empty fields but keeps populated ones",
			mutate: func(r *TransactionRequest) {
				r.CustomerDetail = &CustomerDetail{
					FirstName:   "John",
					PhoneNumber: "08123456789",
				}
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				detail := assertJSONObject(t, body, "customerDetail")
				assertJSONString(t, detail, "firstName", "John")
				assertJSONString(t, detail, "phoneNumber", "08123456789")
				for _, key := range []string{"lastName", "email", "billingAddress", "shippingAddress"} {
					assertJSONAbsent(t, detail, key)
				}
			},
		},
		{
			name: "Address omits empty fields",
			mutate: func(r *TransactionRequest) {
				r.CustomerDetail = &CustomerDetail{
					FirstName: "John",
					BillingAddress: &Address{
						City:        "Jakarta",
						CountryCode: "ID",
					},
				}
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				detail := assertJSONObject(t, body, "customerDetail")
				address := assertJSONObject(t, detail, "billingAddress")
				assertJSONString(t, address, "city", "Jakarta")
				assertJSONString(t, address, "countryCode", "ID")
				for _, key := range []string{"firstName", "lastName", "address", "postalCode", "phone"} {
					assertJSONAbsent(t, address, key)
				}
			},
		},
		{
			name: "itemDetails payload",
			mutate: func(r *TransactionRequest) {
				r.ItemDetails = []ItemDetail{
					{Name: "Product 1", Price: 25000, Quantity: 1},
					{Name: "Product 2", Price: 15000, Quantity: 1},
				}
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				items := assertJSONArray(t, body, "itemDetails")
				if len(items) != 2 {
					t.Fatalf("itemDetails length = %d, want 2", len(items))
				}

				want := []ItemDetail{
					{Name: "Product 1", Price: 25000, Quantity: 1},
					{Name: "Product 2", Price: 15000, Quantity: 1},
				}
				for i, raw := range items {
					var got ItemDetail
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Fatalf("itemDetails[%d] is not an item: %s", i, raw)
					}
					if got != want[i] {
						t.Errorf("itemDetails[%d] = %+v, want %+v", i, got, want[i])
					}
				}
			},
		},
		{
			name: "accountLink OVO payload",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodOVOLink
				r.AccountLink = &AccountLink{
					CredentialCode: "A0F22572-4AF1-E111-812C-B01224449936",
					OVO: &OVODetail{
						PaymentDetails: []OVOPaymentDetail{
							{PaymentType: "CASH", Amount: 40000},
						},
					},
				}
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				link := assertJSONObject(t, body, "accountLink")
				assertJSONString(t, link, "credentialCode", "A0F22572-4AF1-E111-812C-B01224449936")
				assertJSONAbsent(t, link, "shopee")

				ovo := assertJSONObject(t, link, "ovo")
				details := assertJSONArray(t, ovo, "paymentDetails")
				if len(details) != 1 {
					t.Fatalf("paymentDetails length = %d, want 1", len(details))
				}

				var got OVOPaymentDetail
				if err := json.Unmarshal(details[0], &got); err != nil {
					t.Fatalf("paymentDetails[0] is not a payment detail: %s", details[0])
				}
				if got.PaymentType != "CASH" || got.Amount != 40000 {
					t.Errorf("paymentDetails[0] = %+v, want {CASH 40000}", got)
				}
			},
		},
		{
			name: "accountLink Shopee payload",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodShopeeLink
				r.AccountLink = &AccountLink{
					CredentialCode: "7cXXXXX-XXXX-XXXX-9XXX-944XXXXXXX8",
					Shopee: &ShopeeDetail{
						UseCoin: true,
						PromoID: "campaign111",
					},
				}
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				link := assertJSONObject(t, body, "accountLink")
				assertJSONString(t, link, "credentialCode", "7cXXXXX-XXXX-XXXX-9XXX-944XXXXXXX8")
				assertJSONAbsent(t, link, "ovo")

				shopee := assertJSONObject(t, link, "shopee")
				if string(shopee["useCoin"]) != "true" {
					t.Errorf("shopee.useCoin = %s, want true", shopee["useCoin"])
				}
				assertJSONString(t, shopee, "promoId", "campaign111")
			},
		},
		{
			// The docs describe acquirer as a 3 digit bank code such as 014
			// (BCA) or 022 (CIMB), and binWhitelist entries as a 3 digit bank
			// code or a 6 digit card BIN.
			name: "creditCardDetail payload",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodCreditCard
				r.CreditCardDetail = &CreditCardDetail{
					Acquirer:     "014",
					BinWhitelist: []string{"014", "400000"},
				}
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				card := assertJSONObject(t, body, "creditCardDetail")
				assertJSONString(t, card, "acquirer", "014")

				bins := assertJSONArray(t, card, "binWhitelist")
				want := []string{"014", "400000"}
				if len(bins) != len(want) {
					t.Fatalf("binWhitelist length = %d, want %d", len(bins), len(want))
				}
				for i, raw := range bins {
					var got string
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Fatalf("binWhitelist[%d] is not a string: %s", i, raw)
					}
					if got != want[i] {
						t.Errorf("binWhitelist[%d] = %s, want %s", i, got, want[i])
					}
				}
			},
		},
		{
			name: "creditCardDetail omits an empty acquirer",
			mutate: func(r *TransactionRequest) {
				r.CreditCardDetail = &CreditCardDetail{BinWhitelist: []string{"400000"}}
			},
			assert: func(t *testing.T, body map[string]json.RawMessage) {
				card := assertJSONObject(t, body, "creditCardDetail")
				assertJSONAbsent(t, card, "acquirer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, captured := captureServer(t, http.StatusOK, successBody)

			client := newTestClient(server)

			request := baseTransactionRequest()
			tt.mutate(&request)

			if _, err := client.CreateTransaction(request); err != nil {
				t.Fatalf("CreateTransaction() error = %v, want nil", err)
			}

			tt.assert(t, *captured)
		})
	}
}

// TestCreateTransactionSignature covers the inquiry signature across payloads,
// including the documented vector.
func TestCreateTransactionSignature(t *testing.T) {
	const apiKey = "DXXXXCX80TZJ85Q70QCI"

	tests := []struct {
		name            string
		merchantOrderID string
		paymentAmount   int
		expected        string
	}{
		{
			name:            "Documented vector",
			merchantOrderID: "ORDER123",
			paymentAmount:   40000,
			expected:        expectedInquirySignature,
		},
		{
			name:            "Minimum amount",
			merchantOrderID: "ORDER123",
			paymentAmount:   10000,
			expected:        testHMAC(apiKey, "DXXXXORDER12310000"),
		},
		{
			name:            "Longer order id",
			merchantOrderID: "INV-2026-0001",
			paymentAmount:   150000,
			expected:        testHMAC(apiKey, "DXXXXINV-2026-0001150000"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, captured := captureServer(t, http.StatusOK, `{"merchantCode":"DXXXX","amount":"1","statusCode":"00","statusMessage":"SUCCESS"}`)

			client := newTestClient(server)

			request := baseTransactionRequest()
			request.MerchantOrderID = tt.merchantOrderID
			request.PaymentAmount = tt.paymentAmount

			if _, err := client.CreateTransaction(request); err != nil {
				t.Fatalf("CreateTransaction() error = %v, want nil", err)
			}

			body := *captured

			assertJSONString(t, body, "signature", tt.expected)
			if len(tt.expected) != 64 {
				t.Errorf("expected signature length = %d, want 64", len(tt.expected))
			}
			if tt.expected == legacyMD5Signature("DXXXX"+tt.merchantOrderID+strconv.Itoa(tt.paymentAmount), apiKey) {
				t.Error("expected signature matches the obsolete MD5 scheme, want HMAC-SHA256")
			}
		})
	}
}

// TestCreateTransactionErrors covers the documented failure modes of the inquiry
// endpoint.
func TestCreateTransactionErrors(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		responseBody  string
		wantErrSubstr string
	}{
		{
			name:          "Failed statusCode in a 200 response",
			status:        http.StatusOK,
			responseBody:  `{"merchantCode":"DXXXX","amount":"40000","statusCode":"01","statusMessage":"ERROR"}`,
			wantErrSubstr: "error creating transaction: ERROR (statusCode 01)",
		},
		{
			name:          "Wrong signature",
			status:        http.StatusUnauthorized,
			responseBody:  `{"Message":"Wrong signature"}`,
			wantErrSubstr: "duitku: API error (HTTP 401): Wrong signature",
		},
		{
			name:          "Minimum payment",
			status:        http.StatusBadRequest,
			responseBody:  `{"Message":"Minimum Payment 10000 IDR"}`,
			wantErrSubstr: "duitku: API error (HTTP 400): Minimum Payment 10000 IDR",
		},
		{
			name:          "Item total mismatch",
			status:        http.StatusConflict,
			responseBody:  `{"Message":"Payment amount must be equal to all item price"}`,
			wantErrSubstr: "duitku: API error (HTTP 409): Payment amount must be equal to all item price",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := captureServer(t, tt.status, tt.responseBody)

			client := newTestClient(server)

			_, err := client.CreateTransaction(baseTransactionRequest())
			if err == nil {
				t.Fatal("CreateTransaction() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("CreateTransaction() error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

// TestCheckTransaction covers the documented status codes and amount encodings.
func TestCheckTransaction(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		wantStatusCode string
		wantStatusMsg  string
		wantAmount     string
		wantAmountInt  int
	}{
		{
			name: "Success",
			responseBody: `{
				"merchantOrderId": "ORDER123",
				"reference": "DXXXXCX80TZJ85Q70QCI",
				"amount": "100000",
				"fee": "0.00",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			wantStatusCode: CheckTransactionStatusSuccess,
			wantStatusMsg:  "SUCCESS",
			wantAmount:     "100000",
			wantAmountInt:  100000,
		},
		{
			name: "Pending",
			responseBody: `{
				"merchantOrderId": "ORDER123",
				"reference": "DXXXXCX80TZJ85Q70QCI",
				"amount": "100000",
				"fee": "0.00",
				"statusCode": "01",
				"statusMessage": "PENDING"
			}`,
			wantStatusCode: CheckTransactionStatusPending,
			wantStatusMsg:  "PENDING",
			wantAmount:     "100000",
			wantAmountInt:  100000,
		},
		{
			name: "Cancelled",
			responseBody: `{
				"merchantOrderId": "ORDER123",
				"reference": "DXXXXCX80TZJ85Q70QCI",
				"amount": "100000",
				"fee": "0.00",
				"statusCode": "02",
				"statusMessage": "CANCELED"
			}`,
			wantStatusCode: CheckTransactionStatusCancelled,
			wantStatusMsg:  "CANCELED",
			wantAmount:     "100000",
			wantAmountInt:  100000,
		},
		{
			// The documented example quotes amount, but the response table
			// types it as an integer.
			name: "Amount sent as a bare number",
			responseBody: `{
				"merchantOrderId": "ORDER123",
				"reference": "DXXXXCX80TZJ85Q70QCI",
				"amount": 100000,
				"fee": "0.00",
				"statusCode": "00",
				"statusMessage": "SUCCESS"
			}`,
			wantStatusCode: CheckTransactionStatusSuccess,
			wantStatusMsg:  "SUCCESS",
			wantAmount:     "100000",
			wantAmountInt:  100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, captured := captureServer(t, http.StatusOK, tt.responseBody)

			client := newTestClient(server)

			response, err := client.CheckTransaction("ORDER123")
			if err != nil {
				t.Fatalf("CheckTransaction() error = %v, want nil", err)
			}

			body := *captured
			assertJSONString(t, body, "merchantCode", "DXXXX")
			assertJSONString(t, body, "merchantOrderId", "ORDER123")
			assertJSONString(t, body, "signature", expectedCheckSignature)

			if response.MerchantOrderID != "ORDER123" {
				t.Errorf("Response MerchantOrderID = %s, want ORDER123", response.MerchantOrderID)
			}
			if response.Reference != "DXXXXCX80TZJ85Q70QCI" {
				t.Errorf("Response Reference = %s, want DXXXXCX80TZJ85Q70QCI", response.Reference)
			}
			if response.Fee != "0.00" {
				t.Errorf("Response Fee = %s, want 0.00", response.Fee)
			}
			if response.StatusCode != tt.wantStatusCode {
				t.Errorf("Response StatusCode = %s, want %s", response.StatusCode, tt.wantStatusCode)
			}
			if response.StatusMessage != tt.wantStatusMsg {
				t.Errorf("Response StatusMessage = %s, want %s", response.StatusMessage, tt.wantStatusMsg)
			}
			if response.Amount != StringOrInt(tt.wantAmount) {
				t.Errorf("Response Amount = %q, want %q", response.Amount, tt.wantAmount)
			}
			if got, err := response.Amount.Int(); err != nil || got != tt.wantAmountInt {
				t.Errorf("Response Amount.Int() = (%d, %v), want (%d, nil)", got, err, tt.wantAmountInt)
			}
		})
	}
}

// TestCheckTransactionSignature covers the check transaction signature,
// including the documented vector.
func TestCheckTransactionSignature(t *testing.T) {
	const apiKey = "DXXXXCX80TZJ85Q70QCI"

	tests := []struct {
		name            string
		merchantOrderID string
		expected        string
	}{
		{
			name:            "Documented vector",
			merchantOrderID: "ORDER123",
			expected:        expectedCheckSignature,
		},
		{
			name:            "Another order id",
			merchantOrderID: "abcde12345",
			expected:        testHMAC(apiKey, "DXXXXabcde12345"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, captured := captureServer(t, http.StatusOK, `{"merchantOrderId":"ORDER123","amount":"1","statusCode":"00","statusMessage":"SUCCESS"}`)

			client := newTestClient(server)

			if _, err := client.CheckTransaction(tt.merchantOrderID); err != nil {
				t.Fatalf("CheckTransaction() error = %v, want nil", err)
			}

			body := *captured

			assertJSONString(t, body, "signature", tt.expected)
			if tt.expected == legacyMD5Signature("DXXXX"+tt.merchantOrderID, apiKey) {
				t.Error("expected signature matches the obsolete MD5 scheme, want HMAC-SHA256")
			}
		})
	}
}

// TestCheckTransactionErrors covers the failure modes of the check transaction
// endpoint.
func TestCheckTransactionErrors(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		responseBody  string
		wantErrSubstr string
	}{
		{
			name:          "Merchant not found",
			status:        http.StatusNotFound,
			responseBody:  `{"Message":"Merchant not found"}`,
			wantErrSubstr: "duitku: API error (HTTP 404): Merchant not found",
		},
		{
			name:          "Wrong signature",
			status:        http.StatusUnauthorized,
			responseBody:  `{"Message":"Wrong signature"}`,
			wantErrSubstr: "duitku: API error (HTTP 401): Wrong signature",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := captureServer(t, tt.status, tt.responseBody)

			client := newTestClient(server)

			_, err := client.CheckTransaction("ORDER123")
			if err == nil {
				t.Fatal("CheckTransaction() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("CheckTransaction() error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

// TestTransactionEndpoints covers the URL path each method targets.
func TestTransactionEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		call     func(*Client) error
	}{
		{
			name:     "CreateTransaction posts to merchant/v2/inquiry",
			endpoint: "/merchant/v2/inquiry",
			call: func(c *Client) error {
				_, err := c.CreateTransaction(baseTransactionRequest())
				return err
			},
		},
		{
			name:     "CheckTransaction posts to merchant/transactionStatus",
			endpoint: "/merchant/transactionStatus",
			call: func(c *Client) error {
				_, err := c.CheckTransaction("ORDER123")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"merchantCode":"DXXXX","amount":"1","statusCode":"00","statusMessage":"SUCCESS"}`))
			}))
			defer server.Close()

			client := newTestClient(server)

			if err := tt.call(client); err != nil {
				t.Fatalf("call error = %v, want nil", err)
			}
			if gotPath != tt.endpoint {
				t.Errorf("request path = %s, want %s", gotPath, tt.endpoint)
			}
		})
	}
}

// TestJSONOptionsTolerance pins the deliberate leniency of duitkuJSONOptions.
//
// Duitku is a third party gateway whose docs disagree about key casing, so a
// casing difference or a repeated key must never turn a valid payment response
// into a decoding error. These cases would fail under the encoding/json/v2
// defaults, which are case sensitive and reject duplicate members.
func TestJSONOptionsTolerance(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		wantAmount   StringOrInt
	}{
		{
			name:         "Documented casing",
			responseBody: `{"merchantCode":"DXXXX","amount":"40000","statusCode":"00","statusMessage":"SUCCESS"}`,
			wantAmount:   "40000",
		},
		{
			name:         "All lowercase keys",
			responseBody: `{"merchantcode":"DXXXX","amount":"40000","statuscode":"00","statusmessage":"SUCCESS"}`,
			wantAmount:   "40000",
		},
		{
			name:         "Mixed casing keys",
			responseBody: `{"MerchantCode":"DXXXX","AMOUNT":"40000","StatusCode":"00","StatusMessage":"SUCCESS"}`,
			wantAmount:   "40000",
		},
		{
			name:         "Duplicate amount members",
			responseBody: `{"merchantCode":"DXXXX","amount":"40000","amount":"40000","statusCode":"00","statusMessage":"SUCCESS"}`,
			wantAmount:   "40000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := captureServer(t, http.StatusOK, tt.responseBody)

			client := newTestClient(server)

			response, err := client.CreateTransaction(baseTransactionRequest())
			if err != nil {
				t.Fatalf("CreateTransaction() error = %v, want nil", err)
			}

			if response.StatusCode != StatusSuccess {
				t.Errorf("Response StatusCode = %s, want %s", response.StatusCode, StatusSuccess)
			}
			if response.StatusMessage != "SUCCESS" {
				t.Errorf("Response StatusMessage = %s, want SUCCESS", response.StatusMessage)
			}
			if response.Amount != tt.wantAmount {
				t.Errorf("Response Amount = %q, want %q", response.Amount, tt.wantAmount)
			}
		})
	}
}
