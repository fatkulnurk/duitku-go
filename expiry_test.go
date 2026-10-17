package duitku

import (
	"strings"
	"testing"
)

// TestExpiryPeriodFor covers the documented expiry table.
// https://docs.duitku.com/api/id/#expiry-period
func TestExpiryPeriodFor(t *testing.T) {
	const vaDefault = 1440

	tests := []struct {
		name          string
		paymentMethod string
		wantDefault   int
		wantMax       int
		wantFixed     bool
	}{
		{name: "Credit Card", paymentMethod: PaymentMethodCreditCard, wantDefault: 30, wantFixed: true},

		{name: "BCA VA", paymentMethod: PaymentMethodBCA, wantDefault: vaDefault},
		{name: "Mandiri VA", paymentMethod: PaymentMethodMandiri, wantDefault: vaDefault},
		{name: "Maybank VA", paymentMethod: PaymentMethodMaybank, wantDefault: vaDefault},
		{name: "BNI VA", paymentMethod: PaymentMethodBNI, wantDefault: vaDefault},
		{name: "CIMB VA", paymentMethod: PaymentMethodCIMB, wantDefault: vaDefault},
		{name: "Permata VA", paymentMethod: PaymentMethodPermata, wantDefault: vaDefault},
		{name: "ATM Bersama", paymentMethod: PaymentMethodATMBersama, wantDefault: vaDefault},
		{name: "Artha Graha", paymentMethod: PaymentMethodArthaGraha, wantDefault: vaDefault},
		{name: "Neo Commerce", paymentMethod: PaymentMethodNeoCommerce, wantDefault: vaDefault},
		{name: "BRI VA", paymentMethod: PaymentMethodBRI, wantDefault: vaDefault},
		{name: "Sahabat Sampoerna", paymentMethod: PaymentMethodSahabat, wantDefault: vaDefault},
		{name: "Danamon VA", paymentMethod: PaymentMethodDanamon, wantDefault: vaDefault},
		{name: "BSI VA", paymentMethod: PaymentMethodBSI, wantDefault: vaDefault},

		{name: "Pegadaian/ALFA/Pos", paymentMethod: PaymentMethodAlfamart, wantDefault: vaDefault},
		{name: "Indomaret", paymentMethod: PaymentMethodIndomaret, wantDefault: vaDefault},

		{name: "OVO", paymentMethod: PaymentMethodOVO, wantDefault: 10, wantMax: 1440},
		{name: "Shopee Pay Apps", paymentMethod: PaymentMethodShopeePay, wantDefault: 10, wantMax: 60},
		{name: "LinkAja Fixed Fee", paymentMethod: PaymentMethodLinkAjaFixed, wantDefault: 24, wantMax: 1440, wantFixed: true},
		{name: "LinkAja Percentage Fee", paymentMethod: PaymentMethodLinkAjaPercent, wantDefault: 24, wantMax: 1440, wantFixed: true},
		{name: "DANA", paymentMethod: PaymentMethodDANA, wantDefault: 1440, wantMax: 1440},
		{name: "Shopee Account Link", paymentMethod: PaymentMethodShopeeLink, wantDefault: 30, wantFixed: true},
		{name: "OVO Account Link", paymentMethod: PaymentMethodOVOLink, wantDefault: 15, wantFixed: true},

		{name: "Jenius Pay", paymentMethod: PaymentMethodJeniusPay, wantDefault: 10, wantMax: 10},

		{name: "QRIS Shopee Pay", paymentMethod: PaymentMethodQrisShopeePay, wantDefault: 10, wantMax: 60},
		{name: "QRIS Nobu", paymentMethod: PaymentMethodQrisNobu, wantDefault: 24, wantMax: 1440},

		{name: "Indodana Paylater", paymentMethod: PaymentMethodIndodanaPaylater, wantDefault: 1440, wantMax: 1440},
		{name: "ATOME", paymentMethod: PaymentMethodAtome, wantDefault: 720, wantMax: 720},

		{name: "Tokopedia Card", paymentMethod: PaymentMethodTokopediaCard, wantDefault: 1440, wantMax: 1440},
		{name: "Tokopedia E-Wallet", paymentMethod: PaymentMethodTokopediaEWallet, wantDefault: 1440, wantMax: 1440},
		{name: "Tokopedia Others", paymentMethod: PaymentMethodTokopediaOthers, wantDefault: 1440, wantMax: 1440},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExpiryPeriodFor(tt.paymentMethod)
			if !ok {
				t.Fatalf("ExpiryPeriodFor(%s) not found, want the documented limits", tt.paymentMethod)
			}
			if got.Default != tt.wantDefault {
				t.Errorf("Default = %d, want %d", got.Default, tt.wantDefault)
			}
			if got.Max != tt.wantMax {
				t.Errorf("Max = %d, want %d", got.Max, tt.wantMax)
			}
			if got.Fixed != tt.wantFixed {
				t.Errorf("Fixed = %v, want %v", got.Fixed, tt.wantFixed)
			}
		})
	}
}

// TestExpiryPeriodForUnknownMethod covers a method with no published limits.
func TestExpiryPeriodForUnknownMethod(t *testing.T) {
	tests := []struct {
		name          string
		paymentMethod string
	}{
		{name: "QRIS Nusapay has no documented window", paymentMethod: PaymentMethodQrisNusapay},
		{name: "Unknown code", paymentMethod: "ZZ"},
		{name: "Empty code", paymentMethod: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := ExpiryPeriodFor(tt.paymentMethod); ok {
				t.Errorf("ExpiryPeriodFor(%q) found limits, want none", tt.paymentMethod)
			}
		})
	}
}

func TestValidateExpiryPeriod(t *testing.T) {
	tests := []struct {
		name          string
		paymentMethod string
		expiryPeriod  int
		wantErr       bool
		wantErrSubstr string
	}{
		{name: "Zero is always valid", paymentMethod: PaymentMethodQrisShopeePay, expiryPeriod: 0},
		{name: "Negative is treated as unset", paymentMethod: PaymentMethodQrisShopeePay, expiryPeriod: -1},

		{name: "VA default window", paymentMethod: PaymentMethodBCA, expiryPeriod: 1440},
		{name: "VA beyond the documented default is allowed", paymentMethod: PaymentMethodBCA, expiryPeriod: 2880},

		{name: "OVO within max", paymentMethod: PaymentMethodOVO, expiryPeriod: 1440},
		{
			name:          "OVO beyond max",
			paymentMethod: PaymentMethodOVO,
			expiryPeriod:  1441,
			wantErr:       true,
			wantErrSubstr: "exceeds the maximum of 1440 minutes",
		},
		{name: "QRIS Shopee at the limit", paymentMethod: PaymentMethodQrisShopeePay, expiryPeriod: 60},
		{
			name:          "QRIS Shopee beyond the limit",
			paymentMethod: PaymentMethodQrisShopeePay,
			expiryPeriod:  61,
			wantErr:       true,
			wantErrSubstr: "exceeds the maximum of 60 minutes",
		},
		{
			name:          "Jenius Pay beyond its 10 minute limit",
			paymentMethod: PaymentMethodJeniusPay,
			expiryPeriod:  60,
			wantErr:       true,
			wantErrSubstr: "exceeds the maximum of 10 minutes",
		},
		{
			name:          "ATOME beyond 720",
			paymentMethod: PaymentMethodAtome,
			expiryPeriod:  721,
			wantErr:       true,
			wantErrSubstr: "exceeds the maximum of 720 minutes",
		},

		{
			name:          "Credit Card ignores the requested value",
			paymentMethod: PaymentMethodCreditCard,
			expiryPeriod:  60,
			wantErr:       true,
			wantErrSubstr: "always uses the default expiry of 30 minutes",
		},
		{
			name:          "OVO Account Link ignores the requested value",
			paymentMethod: PaymentMethodOVOLink,
			expiryPeriod:  60,
			wantErr:       true,
			wantErrSubstr: "always uses the default expiry of 15 minutes",
		},
		{
			name:          "Shopee Account Link ignores the requested value",
			paymentMethod: PaymentMethodShopeeLink,
			expiryPeriod:  60,
			wantErr:       true,
			wantErrSubstr: "always uses the default expiry of 30 minutes",
		},

		{name: "Unknown method is not validated", paymentMethod: "ZZ", expiryPeriod: 99999},
		{name: "Empty method is not validated", paymentMethod: "", expiryPeriod: 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpiryPeriod(tt.paymentMethod, tt.expiryPeriod)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateExpiryPeriod(%s, %d) error = %v, wantErr %v",
					tt.paymentMethod, tt.expiryPeriod, err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}
