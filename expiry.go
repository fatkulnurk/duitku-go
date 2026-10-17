package duitku

import "fmt"

// ExpiryPeriodLimits describes the documented expiry window for a payment
// method.
// https://docs.duitku.com/api/id/#expiry-period
type ExpiryPeriodLimits struct {
	// Default is the expiry in minutes Duitku applies when expiryPeriod is not
	// sent.
	Default int
	// Max is the largest expiry in minutes Duitku accepts. Zero means the docs
	// publish no maximum.
	Max int
	// Fixed reports that Duitku always uses Default and ignores whatever the
	// merchant requests.
	Fixed bool
}

// expiryPeriodLimits is the documented expiry table, keyed by payment method
// code.
var expiryPeriodLimits = map[string]ExpiryPeriodLimits{
	PaymentMethodCreditCard: {Default: 30, Fixed: true},

	PaymentMethodBCA:         {Default: 1440},
	PaymentMethodMandiri:     {Default: 1440},
	PaymentMethodMaybank:     {Default: 1440},
	PaymentMethodBNI:         {Default: 1440},
	PaymentMethodCIMB:        {Default: 1440},
	PaymentMethodPermata:     {Default: 1440},
	PaymentMethodATMBersama:  {Default: 1440},
	PaymentMethodArthaGraha:  {Default: 1440},
	PaymentMethodNeoCommerce: {Default: 1440},
	PaymentMethodBRI:         {Default: 1440},
	PaymentMethodSahabat:     {Default: 1440},
	PaymentMethodDanamon:     {Default: 1440},
	PaymentMethodBSI:         {Default: 1440},

	PaymentMethodAlfamart:  {Default: 1440},
	PaymentMethodIndomaret: {Default: 1440},

	PaymentMethodOVO:            {Default: 10, Max: 1440},
	PaymentMethodShopeePay:      {Default: 10, Max: 60},
	PaymentMethodLinkAjaFixed:   {Default: 24, Max: 1440, Fixed: true},
	PaymentMethodLinkAjaPercent: {Default: 24, Max: 1440, Fixed: true},
	PaymentMethodDANA:           {Default: 1440, Max: 1440},
	PaymentMethodShopeeLink:     {Default: 30, Fixed: true},
	PaymentMethodOVOLink:        {Default: 15, Fixed: true},

	PaymentMethodJeniusPay: {Default: 10, Max: 10},

	PaymentMethodQrisShopeePay: {Default: 10, Max: 60},
	PaymentMethodQrisNobu:      {Default: 24, Max: 1440},

	PaymentMethodIndodanaPaylater: {Default: 1440, Max: 1440},
	PaymentMethodAtome:            {Default: 720, Max: 720},

	PaymentMethodTokopediaCard:    {Default: 1440, Max: 1440},
	PaymentMethodTokopediaEWallet: {Default: 1440, Max: 1440},
	PaymentMethodTokopediaOthers:  {Default: 1440, Max: 1440},
}

// ExpiryPeriodFor returns the documented expiry limits for a payment method.
// The boolean is false when the method is unknown or its limits are not
// published.
func ExpiryPeriodFor(paymentMethod string) (ExpiryPeriodLimits, bool) {
	limits, ok := expiryPeriodLimits[paymentMethod]
	return limits, ok
}

// ValidateExpiryPeriod checks an expiryPeriod against the documented limits for
// a payment method.
//
// A zero expiryPeriod is always valid: it means "omit the field and let Duitku
// use the channel default".
func ValidateExpiryPeriod(paymentMethod string, expiryPeriod int) error {
	if expiryPeriod <= 0 {
		return nil
	}

	limits, ok := ExpiryPeriodFor(paymentMethod)
	if !ok {
		return nil
	}

	if limits.Fixed {
		return fmt.Errorf("duitku: payment method %s always uses the default expiry of %d minutes and ignores the requested value",
			paymentMethod, limits.Default)
	}

	if limits.Max > 0 && expiryPeriod > limits.Max {
		return fmt.Errorf("duitku: expiryPeriod %d exceeds the maximum of %d minutes for payment method %s",
			expiryPeriod, limits.Max, paymentMethod)
	}

	return nil
}
