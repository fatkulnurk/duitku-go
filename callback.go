package duitku

import (
	"crypto/hmac"
	"errors"
	"fmt"
	"net/http"
)

// CallbackData represents the data received in a Duitku callback.
//
// The callback is delivered as an HTTP POST with Content-Type
// application/x-www-form-urlencoded.
//
// Parameter descriptions:
//   - merchantCode: Merchant code, sent by the Duitku server to inform which project code is on used. Example: DXXXX
//   - amount: Transaction amount. Example: 150000
//   - merchantOrderId: Transaction number from merchant. Example: abcde12345
//   - productDetail: Description about product/service on transaction. Example: Payment example for example merchant
//   - additionalParam: Additional parameters that you send at the beginning of the transaction request.
//   - paymentCode: Payment method code. Example: VC
//   - resultCode: Result code callback notification. 00 - Success, 01 - Failed
//   - merchantUserId: Customer's username or email on your site. Example: your_customer@example.com
//   - reference: Transaction reference number from Duitku. Please keep it for the purposes of recording or tracking transactions. Example: DXXXXCX80TXXX5Q70QCI
//   - signature: HMAC SHA256 of merchantCode + amount + merchantOrderId keyed with the API key. Example: d842db69f70501fe69487b3d957611c2d4e47335f390a5895b0a762a1bf1f1a0
//   - publisherOrderId: Unique transaction payment number from Duitku. Please keep it for the purposes of recording or tracking transactions. Example: MGUHWKJX3M1KMSQN5
//   - spUserHash: Will be sent to your callback if the payment method using ShopeePay(QRIS, App, and Account Link). If this string parameter contains alphabet and numeric, then it might been paid by Shopee itself. Example: xxxyyyzzz
//   - settlementDate: Settlement date estimation information. Format: YYYY-MM-DD. Example: 2023-07-25
//   - issuerCode: QRIS issuer code information. Example: 93600523
//   - customerName: QRIS issuer account identifier. Only sent for Nobu QRIS (NQ) and Nusapay QRIS (SQ). The value may be partially masked. Example: Bam**** Maul***
//
// url: https://docs.duitku.com/api/id/#parameter-callback
type CallbackData struct {
	MerchantCode     string `json:"merchantCode"`
	Amount           string `json:"amount"`
	MerchantOrderID  string `json:"merchantOrderId"`
	ProductDetail    string `json:"productDetail"`
	AdditionalParam  string `json:"additionalParam"`
	PaymentCode      string `json:"paymentCode"`
	ResultCode       string `json:"resultCode"`
	MerchantUserID   string `json:"merchantUserId"`
	Reference        string `json:"reference"`
	Signature        string `json:"signature"`
	PublisherOrderId string `json:"publisherOrderId"`
	SpUserHash       string `json:"spUserHash"`
	SettlementDate   string `json:"settlementDate"`
	IssuerCode       string `json:"issuerCode"`
	CustomerName     string `json:"customerName"`
}

// ParseCallback parses and authenticates the callback data from an HTTP request.
//
// The signature is verified before any field is handed back, so a returned
// CallbackData is known to come from a holder of the API key.
//
// url: https://docs.duitku.com/api/id/#callback
func (c *Client) ParseCallback(r *http.Request) (*CallbackData, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("error parsing form: %w", err)
	}

	// Callbacks arrive in the POST body. PostFormValue is used rather than
	// FormValue so query string parameters cannot be mixed into signed fields.
	callbackData := &CallbackData{
		MerchantCode:     r.PostFormValue("merchantCode"),
		Amount:           r.PostFormValue("amount"),
		MerchantOrderID:  r.PostFormValue("merchantOrderId"),
		ProductDetail:    r.PostFormValue("productDetail"),
		AdditionalParam:  r.PostFormValue("additionalParam"),
		PaymentCode:      r.PostFormValue("paymentCode"),
		ResultCode:       r.PostFormValue("resultCode"),
		MerchantUserID:   r.PostFormValue("merchantUserId"),
		Reference:        r.PostFormValue("reference"),
		Signature:        r.PostFormValue("signature"),
		PublisherOrderId: r.PostFormValue("publisherOrderId"),
		SpUserHash:       r.PostFormValue("spUserHash"),
		SettlementDate:   r.PostFormValue("settlementDate"),
		IssuerCode:       r.PostFormValue("issuerCode"),
		CustomerName:     r.PostFormValue("customerName"),
	}

	// The docs require merchantCode, amount, merchantOrderId and signature.
	if callbackData.MerchantCode == "" ||
		callbackData.Amount == "" ||
		callbackData.MerchantOrderID == "" ||
		callbackData.Signature == "" {
		return nil, errors.New("missing required callback parameters")
	}

	// A callback for another merchant must never be applied to this client.
	if callbackData.MerchantCode != c.config.MerchantCode {
		return nil, errors.New("callback merchant code does not match the configured merchant code")
	}

	if !c.VerifyCallbackSignature(callbackData) {
		return nil, errors.New("invalid callback signature")
	}

	return callbackData, nil
}

// VerifyCallbackSignature verifies the signature of a callback.
//
//	stringToSign = merchantCode + amount + merchantOrderId
//	signature    = HMAC_SHA256(stringToSign, apiKey), lowercase hex
//
// The comparison is constant time.
func (c *Client) VerifyCallbackSignature(data *CallbackData) bool {
	if data == nil || data.Signature == "" {
		return false
	}

	expectedSignature := c.createSignatureHMACSHA256(
		data.MerchantCode,
		data.Amount,
		data.MerchantOrderID,
	)

	return hmac.Equal([]byte(expectedSignature), []byte(data.Signature))
}

// IsSuccessful returns true if the callback indicates a successful payment.
//
// Only resultCode "00" is a success; every other value is treated as not
// successful.
//
// Note: resultCode is not part of the signed string, so a captured callback can
// have its resultCode altered without invalidating the signature. Confirm the
// final state with CheckTransaction before releasing goods or credit.
func (data *CallbackData) IsSuccessful() bool {
	return data.ResultCode == CallbackStatusSuccess
}

// IsFailed returns true if the callback indicates a failed payment.
func (data *CallbackData) IsFailed() bool {
	return data.ResultCode == CallbackStatusFailed
}

// HandleCallback is a helper function to handle Duitku callbacks.
//
// It answers HTTP 200 only when the handler succeeded. Duitku retries a
// callback up to five times when it does not receive a 200, so a handler error
// returns 500 on purpose: a transient failure such as a database outage gets
// retried. The handler error itself is logged, never echoed to the caller.
func (c *Client) HandleCallback(w http.ResponseWriter, r *http.Request, handler func(*CallbackData) error) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	callbackData, err := c.ParseCallback(r)
	if err != nil {
		c.logger.Printf("duitku: rejecting callback: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := handler(callbackData); err != nil {
		c.logger.Printf("duitku: callback handler failed for order %s: %v", callbackData.MerchantOrderID, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
