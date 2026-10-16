package duitku

import (
	"strings"
	"testing"
)

// validTransactionRequest returns a request that passes Validate, used as the
// baseline for the table below.
func validTransactionRequest() TransactionRequest {
	return TransactionRequest{
		PaymentAmount:   40000,
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		Email:           "customer@example.com",
		PaymentMethod:   PaymentMethodBCA,
		CustomerVaName:  "John Doe",
		ReturnURL:       "https://example.com/return",
		CallbackURL:     "https://example.com/callback",
		ExpiryPeriod:    1440,
	}
}

func TestTransactionRequestValidate(t *testing.T) {
	tests := []struct {
		name          string
		mutate        func(*TransactionRequest)
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:   "Baseline is valid",
			mutate: func(r *TransactionRequest) {},
		},
		{
			name:   "Minimum amount is accepted",
			mutate: func(r *TransactionRequest) { r.PaymentAmount = MinimumPaymentAmount },
		},
		{
			name:          "Below the minimum amount",
			mutate:        func(r *TransactionRequest) { r.PaymentAmount = MinimumPaymentAmount - 1 },
			wantErr:       true,
			wantErrSubstr: "paymentAmount must be at least 10000",
		},
		{
			name:          "Zero amount",
			mutate:        func(r *TransactionRequest) { r.PaymentAmount = 0 },
			wantErr:       true,
			wantErrSubstr: "paymentAmount must be at least 10000",
		},
		{
			name:          "Negative amount",
			mutate:        func(r *TransactionRequest) { r.PaymentAmount = -1 },
			wantErr:       true,
			wantErrSubstr: "paymentAmount must be at least 10000",
		},
		{
			name:          "Missing merchantOrderId",
			mutate:        func(r *TransactionRequest) { r.MerchantOrderID = "" },
			wantErr:       true,
			wantErrSubstr: "merchantOrderId is required",
		},
		{
			name:          "merchantOrderId too long",
			mutate:        func(r *TransactionRequest) { r.MerchantOrderID = strings.Repeat("a", 51) },
			wantErr:       true,
			wantErrSubstr: "merchantOrderId must not exceed 50 characters",
		},
		{
			name:   "merchantOrderId at the limit",
			mutate: func(r *TransactionRequest) { r.MerchantOrderID = strings.Repeat("a", 50) },
		},
		{
			name:          "Missing productDetails",
			mutate:        func(r *TransactionRequest) { r.ProductDetails = "" },
			wantErr:       true,
			wantErrSubstr: "productDetails is required",
		},
		{
			name:          "productDetails too long",
			mutate:        func(r *TransactionRequest) { r.ProductDetails = strings.Repeat("a", 256) },
			wantErr:       true,
			wantErrSubstr: "productDetails must not exceed 255 characters",
		},
		{
			name:          "Missing email",
			mutate:        func(r *TransactionRequest) { r.Email = "" },
			wantErr:       true,
			wantErrSubstr: "email is required",
		},
		{
			name:          "Missing paymentMethod",
			mutate:        func(r *TransactionRequest) { r.PaymentMethod = "" },
			wantErr:       true,
			wantErrSubstr: "paymentMethod is required",
		},
		{
			name:          "paymentMethod of the wrong length",
			mutate:        func(r *TransactionRequest) { r.PaymentMethod = "ABC" },
			wantErr:       true,
			wantErrSubstr: "paymentMethod must be a 2 character code",
		},
		{
			name:          "Missing customerVaName",
			mutate:        func(r *TransactionRequest) { r.CustomerVaName = "" },
			wantErr:       true,
			wantErrSubstr: "customerVaName is required",
		},
		{
			name:          "Missing returnUrl",
			mutate:        func(r *TransactionRequest) { r.ReturnURL = "" },
			wantErr:       true,
			wantErrSubstr: "returnUrl is required",
		},
		{
			name:          "Missing callbackUrl",
			mutate:        func(r *TransactionRequest) { r.CallbackURL = "" },
			wantErr:       true,
			wantErrSubstr: "callbackUrl is required",
		},
		{
			name: "Items summing to the payment amount",
			mutate: func(r *TransactionRequest) {
				r.ItemDetails = []ItemDetail{
					{Name: "Item A", Price: 10000, Quantity: 1},
					{Name: "Item B", Price: 10000, Quantity: 3},
				}
			},
		},
		{
			name: "Items that do not sum to the payment amount",
			mutate: func(r *TransactionRequest) {
				r.ItemDetails = []ItemDetail{
					{Name: "Item A", Price: 10000, Quantity: 1},
					{Name: "Item B", Price: 30000, Quantity: 3},
				}
			},
			wantErr:       true,
			wantErrSubstr: "the item total 100000 does not match paymentAmount 40000",
		},
		{
			name: "Item with an empty name",
			mutate: func(r *TransactionRequest) {
				r.ItemDetails = []ItemDetail{{Name: "", Price: 40000, Quantity: 1}}
			},
			wantErr:       true,
			wantErrSubstr: "itemDetails[0].name is required",
		},
		{
			name: "Item with a zero price",
			mutate: func(r *TransactionRequest) {
				r.ItemDetails = []ItemDetail{{Name: "Item A", Price: 0, Quantity: 1}}
			},
			wantErr:       true,
			wantErrSubstr: "itemDetails[0].price must be greater than zero",
		},
		{
			name: "Item with a zero quantity",
			mutate: func(r *TransactionRequest) {
				r.ItemDetails = []ItemDetail{{Name: "Item A", Price: 40000, Quantity: 0}}
			},
			wantErr:       true,
			wantErrSubstr: "itemDetails[0].quantity must be greater than zero",
		},
		{
			name: "Indodana requires customerDetail and itemDetails",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodIndodanaPaylater
			},
			wantErr:       true,
			wantErrSubstr: "customerDetail is required for payment method DN",
		},
		{
			name: "ATOME requires customerDetail",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodAtome
			},
			wantErr:       true,
			wantErrSubstr: "customerDetail is required for payment method AT",
		},
		{
			name: "ATOME with customerDetail but no itemDetails",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodAtome
				r.CustomerDetail = &CustomerDetail{FirstName: "John"}
			},
			wantErr:       true,
			wantErrSubstr: "itemDetails is required for payment method AT",
		},
		{
			name: "ATOME with both required objects",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodAtome
				r.ExpiryPeriod = 720 // ATOME caps expiry at 720 minutes
				r.CustomerDetail = &CustomerDetail{FirstName: "John"}
				r.ItemDetails = []ItemDetail{{Name: "Item A", Price: 40000, Quantity: 1}}
			},
		},
		{
			name: "E-Commerce requires customerVaName",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodTokopediaCard
				r.CustomerVaName = ""
			},
			wantErr:       true,
			wantErrSubstr: "customerVaName is required",
		},
		{
			name: "E-Commerce with customerVaName",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodTokopediaEWallet
			},
		},
		{
			name: "Expiry within the documented window",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodQrisShopeePay
				r.ExpiryPeriod = 60
			},
		},
		{
			name: "Expiry beyond the documented window",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodQrisShopeePay
				r.ExpiryPeriod = 61
			},
			wantErr:       true,
			wantErrSubstr: "exceeds the maximum of 60 minutes",
		},
		{
			name: "Expiry on a channel that ignores it",
			mutate: func(r *TransactionRequest) {
				r.PaymentMethod = PaymentMethodCreditCard
				r.ExpiryPeriod = 60
			},
			wantErr:       true,
			wantErrSubstr: "always uses the default expiry of 30 minutes",
		},
		{
			name: "Unset expiry is always allowed",
			mutate: func(r *TransactionRequest) {
				r.ExpiryPeriod = 0
			},
		},
		{
			// The Duitku docs' own inquiry example breaks the item total rule:
			// 10000*1 + 30000*3 = 100000 against a paymentAmount of 40000.
			name: "Documented example with an inconsistent item total is rejected",
			mutate: func(r *TransactionRequest) {
				r.PaymentAmount = 40000
				r.ItemDetails = []ItemDetail{
					{Name: "Test Item 1", Price: 10000, Quantity: 1},
					{Name: "Test Item 2", Price: 30000, Quantity: 3},
				}
			},
			wantErr:       true,
			wantErrSubstr: "the item total 100000 does not match paymentAmount 40000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validTransactionRequest()
			tt.mutate(&request)

			err := request.Validate()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}
