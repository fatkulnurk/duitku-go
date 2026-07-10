package duitku

import (
	"fmt"
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
	ExpiryPeriod    int    `json:"expiryPeriod"`

	// Optional fields
	AdditionalParam    string              `json:"additionalParam,omitempty"`
	MerchantUserInfo   string              `json:"merchantUserInfo,omitempty"`
	PhoneNumber        string              `json:"phoneNumber,omitempty"`
	ItemDetails        []ItemDetail        `json:"itemDetails,omitempty"`
	CustomerDetail     *CustomerDetail     `json:"customerDetail,omitempty"`
	AccountLink        *AccountLink        `json:"accountLink,omitempty"`
	CreditCardDetail   *CreditCardDetail   `json:"creditCardDetail,omitempty"`
	IsSubscription     *bool               `json:"isSubscription,omitempty"`
	SubscriptionDetail *SubscriptionDetail `json:"subscriptionDetail,omitempty"`
}

// CustomerDetail represents customer details for a transaction
type CustomerDetail struct {
	FirstName       string   `json:"firstName"`
	LastName        string   `json:"lastName"`
	Email           string   `json:"email"`
	PhoneNumber     string   `json:"phoneNumber,omitempty"`
	BillingAddress  *Address `json:"billingAddress,omitempty"`
	ShippingAddress *Address `json:"shippingAddress,omitempty"`
}

// Address represents an address for billing or shipping
type Address struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Address     string `json:"address"`
	City        string `json:"city"`
	PostalCode  string `json:"postalCode"`
	Phone       string `json:"phone"`
	CountryCode string `json:"countryCode"`
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

// ShopeeDetail represents Shopee payment details
type ShopeeDetail struct {
	UseCoin bool   `json:"useCoin"`
	PromoID string `json:"promoId,omitempty"`
}

// CreditCardDetail represents credit card payment details
type CreditCardDetail struct {
	Acquirer     string   `json:"acquirer"`
	BinWhitelist []string `json:"binWhitelist,omitempty"`
}

// SubscriptionDetail represents subscription details for credit card transactions
type SubscriptionDetail struct {
	Description       string `json:"description"`
	FrequencyType     int    `json:"frequencyType"`
	FrequencyInterval int    `json:"frequencyInterval"`
	TotalNoOfCycles   int    `json:"totalNoOfCycles"`
	FirstRunDate      string `json:"firstRunDate,omitempty"`
}

// TransactionResponse represents the response from creating a transaction
// url: https://docs.duitku.com/api/en/#response-parameters
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
	MerchantCode  string `json:"merchantCode"`
	Reference     string `json:"reference"`
	PaymentURL    string `json:"paymentUrl,omitempty"`
	VANumber      string `json:"vaNumber,omitempty"`
	QrString      string `json:"qrString,omitempty"`
	Amount        string `json:"amount"`
	StatusCode    string `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
}

// CreateTransaction creates a new transaction
// url: https://docs.duitku.com/api/en/#request-transaction
func (c *Client) CreateTransaction(request TransactionRequest) (*TransactionResponse, error) {
	// Create signature
	signature := c.createSignatureMD5(c.config.MerchantCode, request.MerchantOrderID, fmt.Sprintf("%d", request.PaymentAmount))

	// Create the full request
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
	err := c.doRequest("POST", "merchant/v2/inquiry", fullRequest, &response)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != "00" {
		return nil, fmt.Errorf("error creating transaction: %s", response.StatusMessage)
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
// url: https://docs.duitku.com/api/en/#check-transaction-response-parameters
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
	MerchantOrderID string `json:"merchantOrderId"`
	Reference       string `json:"reference"`
	Amount          string `json:"amount"`
	Fee             string `json:"fee"`
	StatusCode      string `json:"statusCode"`
	StatusMessage   string `json:"statusMessage"`
}

// CheckTransaction checks the status of a transaction by merchant order ID
// URL: https://docs.duitku.com/api/en/#check-transaction-response-parameters
func (c *Client) CheckTransaction(merchantOrderID string) (*TransactionStatusResponse, error) {
	// Create signature
	signature := c.createSignatureMD5(c.config.MerchantCode, merchantOrderID)

	request := CheckTransactionRequest{
		MerchantCode:    c.config.MerchantCode,
		MerchantOrderID: merchantOrderID,
		Signature:       signature,
	}

	var response TransactionStatusResponse
	err := c.doRequest("POST", "merchant/transactionStatus", request, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// VoidTransactionRequest represents a request to void a transaction
type VoidTransactionRequest struct {
	MerchantCode    string `json:"merchantCode"`
	MerchantOrderID string `json:"merchantOrderId"`
	Reference       string `json:"reference"`
	Signature       string `json:"signature"`
}

// VoidTransactionResponse represents the response from voiding a transaction
// url: https://docs.duitku.com/api/id/#void-transaction
// example response:
//
//	{
//	  "merchantOrderId": "abcde12345",
//	  "reference": "DXXXXCX80TZJ85Q70QCI",
//	  "statusCode": "00",
//	  "statusMessage": "SUCCESS"
//	}
type VoidTransactionResponse struct {
	MerchantOrderID string `json:"merchantOrderId"`
	Reference       string `json:"reference"`
	StatusCode      string `json:"statusCode"`
	StatusMessage   string `json:"statusMessage"`
}

// VoidTransaction voids a transaction by merchant order ID and reference
// URL: https://docs.duitku.com/api/id/#void-transaction
// Only supported for certain payment methods (OVO, ShopeePay, Credit Card, etc.)
func (c *Client) VoidTransaction(merchantOrderID, reference string) (*VoidTransactionResponse, error) {
	// Create signature
	signature := c.createSignatureMD5(c.config.MerchantCode, merchantOrderID, reference)

	request := VoidTransactionRequest{
		MerchantCode:    c.config.MerchantCode,
		MerchantOrderID: merchantOrderID,
		Reference:       reference,
		Signature:       signature,
	}

	var response VoidTransactionResponse
	err := c.doRequest("POST", "merchant/transaction/delete", request, &response)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != "00" {
		return nil, fmt.Errorf("error voiding transaction: %s", response.StatusMessage)
	}

	return &response, nil
}
