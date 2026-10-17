package duitku

import (
	"fmt"
	"strconv"
)

// TransactionRequest represents a request to create a transaction
type TransactionRequest struct {
	// Required fields
	PaymentAmount   int    `json:"paymentAmount"`
	MerchantOrderID string `json:"merchantOrderId"`
	ProductDetails  string `json:"productDetails"`
	Email           string `json:"email"`
	PaymentMethod   string `json:"paymentMethod"`
	CustomerVaName  string `json:"customerVaName"`
	ReturnURL       string `json:"returnUrl"`
	CallbackURL     string `json:"callbackUrl"`

	// Optional fields
	// ExpiryPeriod is the transaction lifetime in minutes. When left at zero it
	// is omitted so Duitku applies the channel default. Each payment method has
	// its own window; see ExpiryPeriodFor.
	//
	// This is omitzero rather than omitempty: under encoding/json/v2 omitempty
	// keeps a zero int, which would send "expiryPeriod":0 and change the
	// meaning from "use the channel default" to "expire immediately".
	ExpiryPeriod     int               `json:"expiryPeriod,omitzero"`
	AdditionalParam  string            `json:"additionalParam,omitempty"`
	MerchantUserInfo string            `json:"merchantUserInfo,omitempty"`
	PhoneNumber      string            `json:"phoneNumber,omitempty"`
	ItemDetails      []ItemDetail      `json:"itemDetails,omitempty"`
	CustomerDetail   *CustomerDetail   `json:"customerDetail,omitempty"`
	AccountLink      *AccountLink      `json:"accountLink,omitempty"`
	CreditCardDetail *CreditCardDetail `json:"creditCardDetail,omitempty"`
}

// CustomerDetail represents customer details for a transaction
type CustomerDetail struct {
	FirstName       string   `json:"firstName,omitempty"`
	LastName        string   `json:"lastName,omitempty"`
	Email           string   `json:"email,omitempty"`
	PhoneNumber     string   `json:"phoneNumber,omitempty"`
	BillingAddress  *Address `json:"billingAddress,omitempty"`
	ShippingAddress *Address `json:"shippingAddress,omitempty"`
}

// Address represents an address for billing or shipping
type Address struct {
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Address     string `json:"address,omitempty"`
	City        string `json:"city,omitempty"`
	PostalCode  string `json:"postalCode,omitempty"`
	Phone       string `json:"phone,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// ItemDetail represents an item in a transaction
type ItemDetail struct {
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

// AccountLink represents account linking details for OVO and Shopee
type AccountLink struct {
	CredentialCode string        `json:"credentialCode"`
	OVO            *OVODetail    `json:"ovo,omitempty"`
	Shopee         *ShopeeDetail `json:"shopee,omitempty"`
}

// OVODetail represents OVO payment details
type OVODetail struct {
	PaymentDetails []OVOPaymentDetail `json:"paymentDetails"`
}

// OVOPaymentDetail represents a payment detail for OVO
type OVOPaymentDetail struct {
	PaymentType string `json:"paymentType"`
	Amount      int    `json:"amount"`
}

// ShopeeDetail represents Shopee payment details.
//
// Note: the Duitku docs contradict themselves on the voucher field name. The
// "Shopee Detail" table and JSON object example call it `promo_ids`, while the
// account link and request transaction examples call it `promoId`. The request
// examples are the ones merchants integrate against, so `promoId` is used here.
// https://docs.duitku.com/api/id/#shopee-detail
type ShopeeDetail struct {
	UseCoin bool   `json:"useCoin"`
	PromoID string `json:"promoId"`
}

// CreditCardDetail represents credit card payment details
type CreditCardDetail struct {
	// Acquirer is a 3 digit bank code, for example "014" for BCA or "022" for
	// CIMB.
	Acquirer string `json:"acquirer,omitempty"`
	// BinWhitelist restricts which cards are accepted. Each entry is a 3 digit
	// bank code or a 6 digit card BIN. The docs give conflicting maximums (25 in
	// the Indonesian page, 15 in the English page), so no client side limit is
	// enforced here.
	BinWhitelist []string `json:"binWhitelist,omitempty"`
}

// TransactionResponse represents the response from creating a transaction
// url: https://docs.duitku.com/api/id/#parameter-respon-request-transaksi
// example response:
//
//	{
//	 "merchantCode": "DXXXX",
//	 "reference": "DXXXXCX80TZJ85Q70QCI",
//	 "paymentUrl": "https://sandbox.duitku.com/topup/topupdirectv2.aspx?ref=BCA7WZ7EIDXXXXWEC",
//	 "vaNumber": "7007014001444348",
//	 "qrString": "00020101021226660014ID.DANA.WWW011893600911002151500102152006170915150010303UME51450015ID.OR.GPNQR.WWW02150000000000000000303UME520454995802ID5911Toko Jualan6013Jakarta Barat61051153062210117LQKI2LPMJQPKCIIS553033605405400006304502A",
//	 "amount": "40000",
//	 "statusCode": "00",
//	 "statusMessage": "SUCCESS"
//	}
type TransactionResponse struct {
	MerchantCode string `json:"merchantCode"`
	Reference    string `json:"reference"`
	PaymentURL   string `json:"paymentUrl,omitempty"`
	VANumber     string `json:"vaNumber,omitempty"`
	QrString     string `json:"qrString,omitempty"`
	// AppURL is the payment link for e-commerce channels (Tokopedia). The docs
	// example spells the key "AppUrl" while the parameter table spells it
	// "appUrl". duitkuJSONOptions enables case-insensitive name matching, so
	// either spelling decodes into this field.
	AppURL        string      `json:"appUrl,omitempty"`
	Amount        StringOrInt `json:"amount"`
	StatusCode    string      `json:"statusCode"`
	StatusMessage string      `json:"statusMessage"`
}

// CreateTransaction creates a new transaction
// url: https://docs.duitku.com/api/id/#permintaan-transaksi
func (c *Client) CreateTransaction(request TransactionRequest) (*TransactionResponse, error) {
	// stringToSign = merchantCode + merchantOrderId + paymentAmount
	signature := c.createSignatureHMACSHA256(
		c.config.MerchantCode,
		request.MerchantOrderID,
		strconv.Itoa(request.PaymentAmount),
	)

	fullRequest := struct {
		TransactionRequest
		MerchantCode string `json:"merchantCode"`
		Signature    string `json:"signature"`
	}{
		TransactionRequest: request,
		MerchantCode:       c.config.MerchantCode,
		Signature:          signature,
	}

	var response TransactionResponse
	if err := c.doRequest("POST", "merchant/v2/inquiry", fullRequest, &response); err != nil {
		return nil, err
	}

	if response.StatusCode != StatusSuccess {
		return nil, fmt.Errorf("error creating transaction: %s (statusCode %s)", response.StatusMessage, response.StatusCode)
	}

	return &response, nil
}

// CheckTransactionRequest represents a request to check a transaction status
type CheckTransactionRequest struct {
	MerchantCode    string `json:"merchantCode"`
	MerchantOrderID string `json:"merchantOrderId"`
	Signature       string `json:"signature"`
}

// TransactionStatusResponse represents the response from checking a transaction status
// url: https://docs.duitku.com/api/id/#parameter-respon-cek-transaksi
// example response:
//
//	{
//	 "merchantOrderId": "abcde12345",
//	 "reference": "DXXXXCX80TZJ85Q70QCI",
//	 "amount": "100000",
//	 "fee":"0.00",
//	 "statusCode": "00",
//	 "statusMessage": "SUCCESS"
//	}
type TransactionStatusResponse struct {
	MerchantOrderID string      `json:"merchantOrderId"`
	Reference       string      `json:"reference"`
	Amount          StringOrInt `json:"amount"`
	Fee             string      `json:"fee"`
	StatusCode      string      `json:"statusCode"`
	StatusMessage   string      `json:"statusMessage"`
}

// CheckTransaction checks the status of a transaction by merchant order ID.
//
// A successful call does not imply a successful payment: statusCode 01
// (pending) and 02 (canceled) are valid answers. Inspect
// TransactionStatusResponse.StatusCode instead of treating a nil error as
// "paid".
//
// Do not poll this endpoint aggressively. Duitku throttles it and can block a
// caller for roughly an hour once the hit limit is reached.
// url: https://docs.duitku.com/api/id/#cek-transaksi
func (c *Client) CheckTransaction(merchantOrderID string) (*TransactionStatusResponse, error) {
	// stringToSign = merchantCode + merchantOrderId
	signature := c.createSignatureHMACSHA256(c.config.MerchantCode, merchantOrderID)

	request := CheckTransactionRequest{
		MerchantCode:    c.config.MerchantCode,
		MerchantOrderID: merchantOrderID,
		Signature:       signature,
	}

	var response TransactionStatusResponse
	if err := c.doRequest("POST", "merchant/transactionStatus", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
