package duitku

import (
	"fmt"
	"strconv"
	"time"
)

// PaymentMethod represents a payment method available in Duitku
type PaymentMethod struct {
	PaymentMethod string `json:"paymentMethod"`
	PaymentName   string `json:"paymentName"`
	PaymentImage  string `json:"paymentImage"`
	TotalFee      string `json:"totalFee"`
}

// PaymentMethodResponse represents the response from the get payment methods endpoint
// url: https://docs.duitku.com/api/id/#parameter-respon-get-payment-method
// example response:
//
//	{
//	   "paymentFee": [
//	       {
//	           "paymentMethod": "VA",
//	           "paymentName": "MAYBANK VA",
//	           "paymentImage": "https://images.duitku.com/hotlink-ok/VA.PNG",
//	           "totalFee": "0"
//	       },
//	       {
//	           "paymentMethod": "BT",
//	           "paymentName": "PERMATA VA",
//	           "paymentImage": "https://images.duitku.com/hotlink-ok/PERMATA.PNG",
//	           "totalFee": "0"
//	       },
//	   ],
//	   "responseCode": "00",
//	   "responseMessage": "SUCCESS"
//	}
type PaymentMethodResponse struct {
	PaymentFee      []PaymentMethod `json:"paymentFee"`
	ResponseCode    string          `json:"responseCode"`
	ResponseMessage string          `json:"responseMessage"`
}

// GetPaymentMethodsRequest represents the request to get payment methods.
//
// This endpoint is the odd one out in the Duitku API: it expects the lowercase
// key `merchantcode` while the inquiry and check endpoints expect `merchantCode`.
type GetPaymentMethodsRequest struct {
	MerchantCode string `json:"merchantcode"`
	Amount       int    `json:"amount"`
	DateTime     string `json:"datetime"`
	Signature    string `json:"signature"`
}

// GetPaymentMethods retrieves the available payment methods for the specified amount
// url: https://docs.duitku.com/api/id/#get-payment-method
func (c *Client) GetPaymentMethods(amount int) ([]PaymentMethod, error) {
	datetime := time.Now().Format("2006-01-02 15:04:05")

	// stringToSign = merchantcode + amount + datetime
	signature := c.createSignatureHMACSHA256(
		c.config.MerchantCode,
		strconv.Itoa(amount),
		datetime,
	)

	request := GetPaymentMethodsRequest{
		MerchantCode: c.config.MerchantCode,
		Amount:       amount,
		DateTime:     datetime,
		Signature:    signature,
	}

	var response PaymentMethodResponse
	if err := c.doRequest("POST", "merchant/paymentmethod/getpaymentmethod", request, &response); err != nil {
		return nil, err
	}

	if response.ResponseCode != StatusSuccess {
		return nil, fmt.Errorf("error getting payment methods: %s (responseCode %s)", response.ResponseMessage, response.ResponseCode)
	}

	return response.PaymentFee, nil
}
