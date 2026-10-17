package duitku

import (
	"errors"
	"fmt"
)

// Validate checks a TransactionRequest against the rules Duitku documents for
// the inquiry endpoint.
//
// It is opt-in: CreateTransaction does not call it automatically. Duitku
// itself rejects invalid requests with HTTP 400, and the checks here exist so a
// caller can fail fast with a clearer message. The rules are taken from
// https://docs.duitku.com/api/id/#parameter-request-transaksi and
// https://docs.duitku.com/api/id/#metode-pembayaran.
func (r TransactionRequest) Validate() error {
	if r.PaymentAmount < MinimumPaymentAmount {
		return fmt.Errorf("duitku: paymentAmount must be at least %d, got %d",
			MinimumPaymentAmount, r.PaymentAmount)
	}

	if r.MerchantOrderID == "" {
		return errors.New("duitku: merchantOrderId is required")
	}
	if len(r.MerchantOrderID) > MaximumMerchantOrderIDLength {
		return fmt.Errorf("duitku: merchantOrderId must not exceed %d characters, got %d",
			MaximumMerchantOrderIDLength, len(r.MerchantOrderID))
	}

	if r.ProductDetails == "" {
		return errors.New("duitku: productDetails is required")
	}
	if len(r.ProductDetails) > MaximumProductDetailsLength {
		return fmt.Errorf("duitku: productDetails must not exceed %d characters, got %d",
			MaximumProductDetailsLength, len(r.ProductDetails))
	}

	if r.Email == "" {
		return errors.New("duitku: email is required")
	}

	if r.PaymentMethod == "" {
		return errors.New("duitku: paymentMethod is required")
	}
	if len(r.PaymentMethod) != 2 {
		return fmt.Errorf("duitku: paymentMethod must be a 2 character code, got %q", r.PaymentMethod)
	}

	if r.CustomerVaName == "" {
		return errors.New("duitku: customerVaName is required")
	}

	if r.ReturnURL == "" {
		return errors.New("duitku: returnUrl is required")
	}
	if r.CallbackURL == "" {
		return errors.New("duitku: callbackUrl is required")
	}

	if err := r.validateItemDetails(); err != nil {
		return err
	}

	if err := r.validateChannelRequirements(); err != nil {
		return err
	}

	return ValidateExpiryPeriod(r.PaymentMethod, r.ExpiryPeriod)
}

// validateItemDetails enforces the per-item fields and the rule that the item
// total equals paymentAmount.
func (r TransactionRequest) validateItemDetails() error {
	if len(r.ItemDetails) == 0 {
		return nil
	}

	total := 0
	for i, item := range r.ItemDetails {
		if item.Name == "" {
			return fmt.Errorf("duitku: itemDetails[%d].name is required", i)
		}
		if item.Price <= 0 {
			return fmt.Errorf("duitku: itemDetails[%d].price must be greater than zero, got %d", i, item.Price)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("duitku: itemDetails[%d].quantity must be greater than zero, got %d", i, item.Quantity)
		}
		total += item.Price * item.Quantity
	}

	if total != r.PaymentAmount {
		return fmt.Errorf("duitku: the item total %d does not match paymentAmount %d",
			total, r.PaymentAmount)
	}

	return nil
}

// validateChannelRequirements enforces the per-channel parameter requirements.
//
// customerVaName is required for every channel by the inquiry parameter table,
// so it is checked unconditionally in Validate rather than here. The docs call
// it out again for e-commerce channels, which is already covered.
func (r TransactionRequest) validateChannelRequirements() error {
	switch r.PaymentMethod {
	case PaymentMethodIndodanaPaylater, PaymentMethodAtome:
		if r.CustomerDetail == nil {
			return fmt.Errorf("duitku: customerDetail is required for payment method %s", r.PaymentMethod)
		}
		if len(r.ItemDetails) == 0 {
			return fmt.Errorf("duitku: itemDetails is required for payment method %s", r.PaymentMethod)
		}
	}

	return nil
}
