package duitku

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTransaction(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request body
		var request struct {
			MerchantCode    string `json:"merchantCode"`
			PaymentAmount   int    `json:"paymentAmount"`
			PaymentMethod   string `json:"paymentMethod"`
			MerchantOrderID string `json:"merchantOrderId"`
			ProductDetails  string `json:"productDetails"`
			Email           string `json:"email"`
			Signature       string `json:"signature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check request fields
		if request.MerchantCode != "DXXXX" {
			t.Errorf("Expected MerchantCode: DXXXX, got %s", request.MerchantCode)
		}
		if request.PaymentAmount != 40000 {
			t.Errorf("Expected PaymentAmount: 40000, got %d", request.PaymentAmount)
		}
		if request.PaymentMethod != "VC" {
			t.Errorf("Expected PaymentMethod: VC, got %s", request.PaymentMethod)
		}
		if request.MerchantOrderID != "ORDER123" {
			t.Errorf("Expected MerchantOrderID: ORDER123, got %s", request.MerchantOrderID)
		}
		if request.ProductDetails != "Test Product" {
			t.Errorf("Expected ProductDetails: Test Product, got %s", request.ProductDetails)
		}
		if request.Email != "customer@example.com" {
			t.Errorf("Expected Email: customer@example.com, got %s", request.Email)
		}
		if request.Signature == "" {
			t.Errorf("Expected Signature to be non-empty")
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "DEV123456789",
			"paymentUrl": "https://sandbox.duitku.com/payment/DEV123456789",
			"vaNumber": "123456789",
			"amount": "40000",
			"statusCode": "00",
			"statusMessage": "SUCCESS"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   "VC",
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		CustomerVaName:  "John Doe",
		Email:           "customer@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    10,
	}

	// Test creating a transaction
	response, err := client.CreateTransaction(request)
	if err != nil {
		t.Errorf("CreateTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.MerchantCode != "DXXXX" {
		t.Errorf("Response MerchantCode = %s, want DXXXX", response.MerchantCode)
	}
	if response.Reference != "DEV123456789" {
		t.Errorf("Response Reference = %s, want DEV123456789", response.Reference)
	}
	if response.PaymentURL != "https://sandbox.duitku.com/payment/DEV123456789" {
		t.Errorf("Response PaymentURL = %s, want https://sandbox.duitku.com/payment/DEV123456789", response.PaymentURL)
	}
	if response.VANumber != "123456789" {
		t.Errorf("Response VANumber = %s, want 123456789", response.VANumber)
	}
	if response.Amount != "40000" {
		t.Errorf("Response Amount = %s, want 40000", response.Amount)
	}
	if response.StatusCode != "00" {
		t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
	}
	if response.StatusMessage != "SUCCESS" {
		t.Errorf("Response StatusMessage = %s, want SUCCESS", response.StatusMessage)
	}
}

func TestCreateTransactionError(t *testing.T) {
	// Create a test server that returns an error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "",
			"paymentUrl": "",
			"amount": 40000,
			"statusCode": "01",
			"statusMessage": "ERROR"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   "VC",
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		CustomerVaName:  "John Doe",
		Email:           "customer@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    10,
	}

	// Test creating a transaction with error
	_, err := client.CreateTransaction(request)
	if err == nil {
		t.Errorf("CreateTransaction() error = nil, want error")
	}
}

func TestCheckTransaction(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request body
		var request CheckTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check request fields
		if request.MerchantCode != "DXXXX" {
			t.Errorf("Expected MerchantCode: DXXXX, got %s", request.MerchantCode)
		}
		if request.MerchantOrderID != "ORDER123" {
			t.Errorf("Expected MerchantOrderID: ORDER123, got %s", request.MerchantOrderID)
		}
		if request.Signature == "" {
			t.Errorf("Expected Signature to be non-empty")
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantOrderId": "ORDER123",
			"reference": "DEV123456789",
			"amount": "40000",
			"fee": "2000",
			"statusCode": "00",
			"statusMessage": "SUCCESS"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Test checking a transaction
	response, err := client.CheckTransaction("ORDER123")
	if err != nil {
		t.Errorf("CheckTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.MerchantOrderID != "ORDER123" {
		t.Errorf("Response MerchantOrderID = %s, want ORDER123", response.MerchantOrderID)
	}
	if response.Reference != "DEV123456789" {
		t.Errorf("Response Reference = %s, want DEV123456789", response.Reference)
	}
	if response.Amount != "40000" {
		t.Errorf("Response Amount = %s, want 40000", response.Amount)
	}
	if response.Fee != "2000" {
		t.Errorf("Response Fee = %s, want 2000", response.Fee)
	}
	if response.StatusCode != "00" {
		t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
	}
	if response.StatusMessage != "SUCCESS" {
		t.Errorf("Response StatusMessage = %s, want SUCCESS", response.StatusMessage)
	}
}

func TestCreateTransactionWithError(t *testing.T) {
	// Create a test server that returns an error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return an error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "",
			"paymentUrl": "",
			"amount": "40000",
			"statusCode": "01",
			"statusMessage": "FAILED"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   PaymentMethodCreditCard,
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		CustomerVaName:  "John Doe",
		Email:           "john@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
	}

	// Test creating a transaction with error response
	_, err := client.CreateTransaction(request)

	// Should return an error
	if err == nil {
		t.Errorf("CreateTransaction() expected error, got nil")
	}

	// Check the error message
	expectedErrMsg := "error creating transaction: FAILED"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestCreateTransactionWithServerError(t *testing.T) {
	// Create a test server that returns a server error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a server error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"500","message":"Internal Server Error"}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   PaymentMethodCreditCard,
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		CustomerVaName:  "John Doe",
		Email:           "john@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
	}

	// Test creating a transaction with server error
	_, err := client.CreateTransaction(request)
	if err == nil {
		t.Errorf("CreateTransaction() expected error, got nil")
	}
}

func TestCheckTransactionWithError(t *testing.T) {
	// Create a test server that returns an error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return an error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantOrderId": "ORDER123",
			"reference": "DEV123456789",
			"amount": "40000",
			"fee": "2000",
			"statusCode": "02",
			"statusMessage": "CANCELLED"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Test checking a transaction with error status
	response, err := client.CheckTransaction("ORDER123")
	if err != nil {
		t.Errorf("CheckTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.StatusCode != "02" {
		t.Errorf("Response StatusCode = %s, want 02", response.StatusCode)
	}
	if response.StatusMessage != "CANCELLED" {
		t.Errorf("Response StatusMessage = %s, want CANCELLED", response.StatusMessage)
	}
}

func TestCheckTransactionWithServerError(t *testing.T) {
	// Create a test server that returns a server error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/merchant/transactionStatus" {
			t.Errorf("Expected path /merchant/transactionStatus, got %s", r.URL.Path)
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		// Return a server error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"statusCode":"500","statusMessage":"Internal Server Error"}`))
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := NewClient(Config{
		MerchantCode: "DXXXX",
		APIKey:       "DXXXXCX80TZJ85Q70QCI",
		IsSandbox:    true,
	})
	// Override the base URL to use the test server
	client.baseURL = server.URL

	// Test checking a transaction with server error
	_, err := client.CheckTransaction("ORDER123")

	// Should return an error
	if err == nil {
		t.Errorf("CheckTransaction() expected error, got nil")
	}

	// We just need to verify that an error was returned, the exact message is not important
	// as long as it's not nil, since the server returned an error status code
}

func TestCheckTransactionWithInvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := NewClient(Config{
		MerchantCode: "DXXXX",
		APIKey:       "DXXXXCX80TZJ85Q70QCI",
		IsSandbox:    true,
	})
	// Override the base URL to use the test server
	client.baseURL = server.URL

	// Test checking a transaction with invalid JSON response
	_, err := client.CheckTransaction("ORDER123")

	// Should return an error
	if err == nil {
		t.Errorf("CheckTransaction() expected error for invalid JSON, got nil")
	}

	// Error should be about JSON parsing or decoding
	if !strings.Contains(err.Error(), "JSON") &&
		!strings.Contains(err.Error(), "json") &&
		!strings.Contains(err.Error(), "decoding") &&
		!strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Expected error about JSON parsing or decoding, got: %v", err)
	}
}

func TestCreateTransactionWithOVOLink(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request body to verify account link fields
		var request struct {
			MerchantCode string       `json:"merchantCode"`
			AccountLink  *AccountLink `json:"accountLink"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check account link fields
		if request.AccountLink == nil {
			t.Errorf("Expected AccountLink to be non-nil")
		} else {
			if request.AccountLink.CredentialCode != "OVO-CREDENTIAL" {
				t.Errorf("Expected CredentialCode: OVO-CREDENTIAL, got %s", request.AccountLink.CredentialCode)
			}
			if request.AccountLink.OVO == nil {
				t.Errorf("Expected OVO to be non-nil")
			} else {
				if len(request.AccountLink.OVO.PaymentDetails) != 1 {
					t.Errorf("Expected 1 payment detail, got %d", len(request.AccountLink.OVO.PaymentDetails))
				} else {
					if request.AccountLink.OVO.PaymentDetails[0].PaymentType != "CASH" {
						t.Errorf("Expected PaymentType: CASH, got %s", request.AccountLink.OVO.PaymentDetails[0].PaymentType)
					}
					if request.AccountLink.OVO.PaymentDetails[0].Amount != 40000 {
						t.Errorf("Expected Amount: 40000, got %d", request.AccountLink.OVO.PaymentDetails[0].Amount)
					}
				}
			}
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "OVO123456789",
			"paymentUrl": "https://sandbox.duitku.com/payment/OVO123456789",
			"amount": "40000",
			"statusCode": "00",
			"statusMessage": "SUCCESS"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request with OVO account link
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   PaymentMethodOVOLink,
		MerchantOrderID: "OVO-ORDER-123",
		ProductDetails:  "Test Product with OVO Link",
		CustomerVaName:  "John Doe",
		Email:           "john@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
		AccountLink: &AccountLink{
			CredentialCode: "OVO-CREDENTIAL",
			OVO: &OVODetail{
				PaymentDetails: []OVOPaymentDetail{
					{
						PaymentType: "CASH",
						Amount:      40000,
					},
				},
			},
		},
	}

	// Test creating a transaction with OVO account link
	response, err := client.CreateTransaction(request)
	if err != nil {
		t.Errorf("CreateTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.Reference != "OVO123456789" {
		t.Errorf("Response Reference = %s, want OVO123456789", response.Reference)
	}
	if response.StatusCode != "00" {
		t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
	}
}

func TestCreateTransactionWithShopeeLink(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request body to verify account link fields
		var request struct {
			MerchantCode string       `json:"merchantCode"`
			AccountLink  *AccountLink `json:"accountLink"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check account link fields
		if request.AccountLink == nil {
			t.Errorf("Expected AccountLink to be non-nil")
		} else {
			if request.AccountLink.CredentialCode != "SHOPEE-CREDENTIAL" {
				t.Errorf("Expected CredentialCode: SHOPEE-CREDENTIAL, got %s", request.AccountLink.CredentialCode)
			}
			if request.AccountLink.Shopee == nil {
				t.Errorf("Expected Shopee to be non-nil")
			} else {
				if !request.AccountLink.Shopee.UseCoin {
					t.Errorf("Expected UseCoin to be true")
				}
				if request.AccountLink.Shopee.PromoID != "PROMO123" {
					t.Errorf("Expected PromoID: PROMO123, got %s", request.AccountLink.Shopee.PromoID)
				}
			}
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "SHOPEE123456789",
			"paymentUrl": "https://sandbox.duitku.com/payment/SHOPEE123456789",
			"amount": "40000",
			"statusCode": "00",
			"statusMessage": "SUCCESS"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request with Shopee account link
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   PaymentMethodShopeeLink,
		MerchantOrderID: "SHOPEE-ORDER-123",
		ProductDetails:  "Test Product with Shopee Link",
		CustomerVaName:  "John Doe",
		Email:           "john@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
		AccountLink: &AccountLink{
			CredentialCode: "SHOPEE-CREDENTIAL",
			Shopee: &ShopeeDetail{
				UseCoin: true,
				PromoID: "PROMO123",
			},
		},
	}

	// Test creating a transaction with Shopee account link
	response, err := client.CreateTransaction(request)
	if err != nil {
		t.Errorf("CreateTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.Reference != "SHOPEE123456789" {
		t.Errorf("Response Reference = %s, want SHOPEE123456789", response.Reference)
	}
	if response.StatusCode != "00" {
		t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
	}
}

func TestCreateTransactionWithCreditCardDetails(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request body to verify credit card details
		var request struct {
			MerchantCode     string            `json:"merchantCode"`
			CreditCardDetail *CreditCardDetail `json:"creditCardDetail"`
			CustomerDetail   *CustomerDetail   `json:"customerDetail"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check credit card details
		if request.CreditCardDetail == nil {
			t.Errorf("Expected CreditCardDetail to be non-nil")
		} else {
			if request.CreditCardDetail.Acquirer != "BNI" {
				t.Errorf("Expected Acquirer: BNI, got %s", request.CreditCardDetail.Acquirer)
			}
			if len(request.CreditCardDetail.BinWhitelist) != 2 {
				t.Errorf("Expected 2 bin whitelist entries, got %d", len(request.CreditCardDetail.BinWhitelist))
			} else {
				if request.CreditCardDetail.BinWhitelist[0] != "411111" {
					t.Errorf("Expected BinWhitelist[0]: 411111, got %s", request.CreditCardDetail.BinWhitelist[0])
				}
				if request.CreditCardDetail.BinWhitelist[1] != "511111" {
					t.Errorf("Expected BinWhitelist[1]: 511111, got %s", request.CreditCardDetail.BinWhitelist[1])
				}
			}
		}

		// Check customer details
		if request.CustomerDetail == nil {
			t.Errorf("Expected CustomerDetail to be non-nil")
		} else {
			if request.CustomerDetail.FirstName != "John" {
				t.Errorf("Expected FirstName: John, got %s", request.CustomerDetail.FirstName)
			}
			if request.CustomerDetail.LastName != "Doe" {
				t.Errorf("Expected LastName: Doe, got %s", request.CustomerDetail.LastName)
			}
			if request.CustomerDetail.Email != "john@example.com" {
				t.Errorf("Expected Email: john@example.com, got %s", request.CustomerDetail.Email)
			}
			if request.CustomerDetail.PhoneNumber != "08123456789" {
				t.Errorf("Expected PhoneNumber: 08123456789, got %s", request.CustomerDetail.PhoneNumber)
			}

			// Check billing address
			if request.CustomerDetail.BillingAddress == nil {
				t.Errorf("Expected BillingAddress to be non-nil")
			} else {
				if request.CustomerDetail.BillingAddress.FirstName != "John" {
					t.Errorf("Expected BillingAddress.FirstName: John, got %s", request.CustomerDetail.BillingAddress.FirstName)
				}
				if request.CustomerDetail.BillingAddress.LastName != "Doe" {
					t.Errorf("Expected BillingAddress.LastName: Doe, got %s", request.CustomerDetail.BillingAddress.LastName)
				}
				if request.CustomerDetail.BillingAddress.Address != "123 Main St" {
					t.Errorf("Expected BillingAddress.Address: 123 Main St, got %s", request.CustomerDetail.BillingAddress.Address)
				}
				if request.CustomerDetail.BillingAddress.City != "Jakarta" {
					t.Errorf("Expected BillingAddress.City: Jakarta, got %s", request.CustomerDetail.BillingAddress.City)
				}
				if request.CustomerDetail.BillingAddress.PostalCode != "12345" {
					t.Errorf("Expected BillingAddress.PostalCode: 12345, got %s", request.CustomerDetail.BillingAddress.PostalCode)
				}
				if request.CustomerDetail.BillingAddress.Phone != "08123456789" {
					t.Errorf("Expected BillingAddress.Phone: 08123456789, got %s", request.CustomerDetail.BillingAddress.Phone)
				}
				if request.CustomerDetail.BillingAddress.CountryCode != "ID" {
					t.Errorf("Expected BillingAddress.CountryCode: ID, got %s", request.CustomerDetail.BillingAddress.CountryCode)
				}
			}
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "CC123456789",
			"paymentUrl": "https://sandbox.duitku.com/payment/CC123456789",
			"amount": "40000",
			"statusCode": "00",
			"statusMessage": "SUCCESS"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request with credit card details and customer details
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   PaymentMethodCreditCard,
		MerchantOrderID: "CC-ORDER-123",
		ProductDetails:  "Test Product with Credit Card",
		CustomerVaName:  "John Doe",
		Email:           "john@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
		CreditCardDetail: &CreditCardDetail{
			Acquirer:     "BNI",
			BinWhitelist: []string{"411111", "511111"},
		},
		CustomerDetail: &CustomerDetail{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "john@example.com",
			PhoneNumber: "08123456789",
			BillingAddress: &Address{
				FirstName:   "John",
				LastName:    "Doe",
				Address:     "123 Main St",
				City:        "Jakarta",
				PostalCode:  "12345",
				Phone:       "08123456789",
				CountryCode: "ID",
			},
		},
	}

	// Test creating a transaction with credit card details
	response, err := client.CreateTransaction(request)
	if err != nil {
		t.Errorf("CreateTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.Reference != "CC123456789" {
		t.Errorf("Response Reference = %s, want CC123456789", response.Reference)
	}
	if response.StatusCode != "00" {
		t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
	}
}

func TestCreateTransactionWithItemDetails(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request body to verify item details
		var request struct {
			MerchantCode string       `json:"merchantCode"`
			ItemDetails  []ItemDetail `json:"itemDetails"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check item details
		if len(request.ItemDetails) != 2 {
			t.Errorf("Expected 2 item details, got %d", len(request.ItemDetails))
		} else {
			// Check first item
			if request.ItemDetails[0].Name != "Product 1" {
				t.Errorf("Expected ItemDetails[0].Name: Product 1, got %s", request.ItemDetails[0].Name)
			}
			if request.ItemDetails[0].Price != 25000 {
				t.Errorf("Expected ItemDetails[0].Price: 25000, got %d", request.ItemDetails[0].Price)
			}
			if request.ItemDetails[0].Quantity != 1 {
				t.Errorf("Expected ItemDetails[0].Quantity: 1, got %d", request.ItemDetails[0].Quantity)
			}

			// Check second item
			if request.ItemDetails[1].Name != "Product 2" {
				t.Errorf("Expected ItemDetails[1].Name: Product 2, got %s", request.ItemDetails[1].Name)
			}
			if request.ItemDetails[1].Price != 15000 {
				t.Errorf("Expected ItemDetails[1].Price: 15000, got %d", request.ItemDetails[1].Price)
			}
			if request.ItemDetails[1].Quantity != 1 {
				t.Errorf("Expected ItemDetails[1].Quantity: 1, got %d", request.ItemDetails[1].Quantity)
			}
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "ITEM123456789",
			"paymentUrl": "https://sandbox.duitku.com/payment/ITEM123456789",
			"amount": "40000",
			"statusCode": "00",
			"statusMessage": "SUCCESS"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a transaction request with item details
	request := TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   PaymentMethodCreditCard,
		MerchantOrderID: "ITEM-ORDER-123",
		ProductDetails:  "Test Product with Item Details",
		CustomerVaName:  "John Doe",
		Email:           "john@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
		ItemDetails: []ItemDetail{
			{
				Name:     "Product 1",
				Price:    25000,
				Quantity: 1,
			},
			{
				Name:     "Product 2",
				Price:    15000,
				Quantity: 1,
			},
		},
	}

	// Test creating a transaction with item details
	response, err := client.CreateTransaction(request)
	if err != nil {
		t.Errorf("CreateTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.Reference != "ITEM123456789" {
		t.Errorf("Response Reference = %s, want ITEM123456789", response.Reference)
	}
	if response.StatusCode != "00" {
		t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
	}
}

func TestCreateSubscriptionTransaction(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request body to verify subscription fields
		var request struct {
			MerchantCode       string              `json:"merchantCode"`
			PaymentAmount      int                 `json:"paymentAmount"`
			PaymentMethod      string              `json:"paymentMethod"`
			MerchantOrderID    string              `json:"merchantOrderId"`
			IsSubscription     bool                `json:"isSubscription"`
			SubscriptionDetail *SubscriptionDetail `json:"subscriptionDetail"`
			Signature          string              `json:"signature"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check subscription fields
		if !request.IsSubscription {
			t.Errorf("Expected IsSubscription to be true")
		}

		if request.SubscriptionDetail == nil {
			t.Errorf("Expected SubscriptionDetail to be non-nil")
		} else {
			if request.SubscriptionDetail.FrequencyType != FrequencyMonthly {
				t.Errorf("Expected FrequencyType: %d, got %d", FrequencyMonthly, request.SubscriptionDetail.FrequencyType)
			}
			if request.SubscriptionDetail.FrequencyInterval != 1 {
				t.Errorf("Expected FrequencyInterval: 1, got %d", request.SubscriptionDetail.FrequencyInterval)
			}
			if request.SubscriptionDetail.TotalNoOfCycles != 12 {
				t.Errorf("Expected TotalNoOfCycles: 12, got %d", request.SubscriptionDetail.TotalNoOfCycles)
			}
			if request.SubscriptionDetail.Description != "Monthly Subscription" {
				t.Errorf("Expected Description: Monthly Subscription, got %s", request.SubscriptionDetail.Description)
			}
		}

		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"merchantCode": "DXXXX",
			"reference": "SUB123456789",
			"paymentUrl": "https://sandbox.duitku.com/payment/SUB123456789",
			"amount": "50000",
			"statusCode": "00",
			"statusMessage": "SUCCESS"
		}`))
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := &Client{
		config: Config{
			MerchantCode: "DXXXX",
			APIKey:       "DXXXXCX80TZJ85Q70QCI",
		},
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Create a subscription transaction request
	isSubscription := true
	request := TransactionRequest{
		PaymentAmount:   50000,
		PaymentMethod:   PaymentMethodCreditCard,
		MerchantOrderID: "SUB-ORDER-123",
		ProductDetails:  "Premium Subscription",
		CustomerVaName:  "Jane Doe",
		Email:           "subscriber@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
		IsSubscription:  &isSubscription,
		SubscriptionDetail: &SubscriptionDetail{
			Description:       "Monthly Subscription",
			FrequencyType:     FrequencyMonthly,
			FrequencyInterval: 1,
			TotalNoOfCycles:   12,
		},
	}

	// Test creating a subscription transaction
	response, err := client.CreateTransaction(request)
	if err != nil {
		t.Errorf("CreateTransaction() error = %v, want nil", err)
	}

	// Check the response
	if response.MerchantCode != "DXXXX" {
		t.Errorf("Response MerchantCode = %s, want DXXXX", response.MerchantCode)
	}
	if response.Reference != "SUB123456789" {
		t.Errorf("Response Reference = %s, want SUB123456789", response.Reference)
	}
	if response.PaymentURL != "https://sandbox.duitku.com/payment/SUB123456789" {
		t.Errorf("Response PaymentURL = %s, want https://sandbox.duitku.com/payment/SUB123456789", response.PaymentURL)
	}
	if response.Amount != "50000" {
		t.Errorf("Response Amount = %s, want 50000", response.Amount)
	}
	if response.StatusCode != "00" {
		t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
	}
	if response.StatusMessage != "SUCCESS" {
		t.Errorf("Response StatusMessage = %s, want SUCCESS", response.StatusMessage)
	}
}

func TestVoidTransaction(t *testing.T) {
// Create a test server that returns a successful response
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// Check request method
if r.Method != "POST" {
t.Errorf("Expected POST request, got %s", r.Method)
}

// Check path
if r.URL.Path != "/merchant/transaction/delete" {
t.Errorf("Expected path /merchant/transaction/delete, got %s", r.URL.Path)
}

// Check content type
if r.Header.Get("Content-Type") != "application/json" {
t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
}

// Decode request body
var request struct {
MerchantCode    string `json:"merchantCode"`
MerchantOrderID string `json:"merchantOrderId"`
Reference       string `json:"reference"`
Signature       string `json:"signature"`
}
if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
t.Errorf("Error decoding request body: %v", err)
http.Error(w, "Bad request", http.StatusBadRequest)
return
}

// Check request fields
if request.MerchantCode != "DXXXX" {
t.Errorf("Expected MerchantCode: DXXXX, got %s", request.MerchantCode)
}
if request.MerchantOrderID != "ORDER123" {
t.Errorf("Expected MerchantOrderID: ORDER123, got %s", request.MerchantOrderID)
}
if request.Reference != "DEV123456789" {
t.Errorf("Expected Reference: DEV123456789, got %s", request.Reference)
}
if request.Signature == "" {
t.Errorf("Expected Signature to be non-empty")
}

// Return a successful response
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write([]byte(`{
"merchantOrderId": "ORDER123",
"reference": "DEV123456789",
"statusCode": "00",
"statusMessage": "SUCCESS"
}`))
}))
defer server.Close()

// Create a client that uses the test server
client := &Client{
config: Config{
MerchantCode: "DXXXX",
APIKey:       "DXXXXCX80TZJ85Q70QCI",
},
baseURL:    server.URL,
httpClient: server.Client(),
}

// Test voiding a transaction
response, err := client.VoidTransaction("ORDER123", "DEV123456789")
if err != nil {
t.Errorf("VoidTransaction() error = %v, want nil", err)
}

// Check the response
if response.MerchantOrderID != "ORDER123" {
t.Errorf("Response MerchantOrderID = %s, want ORDER123", response.MerchantOrderID)
}
if response.Reference != "DEV123456789" {
t.Errorf("Response Reference = %s, want DEV123456789", response.Reference)
}
if response.StatusCode != "00" {
t.Errorf("Response StatusCode = %s, want 00", response.StatusCode)
}
if response.StatusMessage != "SUCCESS" {
t.Errorf("Response StatusMessage = %s, want SUCCESS", response.StatusMessage)
}
}

func TestVoidTransactionError(t *testing.T) {
// Create a test server that returns an error response
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write([]byte(`{
"merchantOrderId": "ORDER123",
"reference": "DEV123456789",
"statusCode": "01",
"statusMessage": "FAILED"
}`))
}))
defer server.Close()

// Create a client that uses the test server
client := &Client{
config: Config{
MerchantCode: "DXXXX",
APIKey:       "DXXXXCX80TZJ85Q70QCI",
},
baseURL:    server.URL,
httpClient: server.Client(),
}

// Test voiding a transaction with error
_, err := client.VoidTransaction("ORDER123", "DEV123456789")
if err == nil {
t.Errorf("VoidTransaction() error = nil, want error")
}

// Check the error message
expectedErrMsg := "error voiding transaction: FAILED"
if err.Error() != expectedErrMsg {
t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
}
}

func TestVoidTransactionWithServerError(t *testing.T) {
// Create a test server that returns a server error
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusInternalServerError)
w.Write([]byte(`{"responseCode":"500","responseMessage":"Internal Server Error"}`))
}))
defer server.Close()

// Create a client that uses the test server
client := &Client{
config: Config{
MerchantCode: "DXXXX",
APIKey:       "DXXXXCX80TZJ85Q70QCI",
},
baseURL:    server.URL,
httpClient: server.Client(),
}

// Test voiding a transaction with server error
_, err := client.VoidTransaction("ORDER123", "DEV123456789")
if err == nil {
t.Errorf("VoidTransaction() expected error, got nil")
}
}
