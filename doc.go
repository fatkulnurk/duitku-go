/*
Package duitku provides a Go client for the Duitku Payment Gateway API.

Duitku is an Indonesian payment gateway that supports various payment methods
including virtual accounts, e-wallets, retail outlets, credit cards, and more.

This package implements Duitku API v2 with the HMAC-SHA256 signature scheme.
The older MD5 and plain SHA256 signature methods are obsolete and are not
implemented: https://docs.duitku.com/api/id/

# Basic Usage

Initialize a client with your merchant code and API key:

	client := duitku.NewClient(duitku.Config{
		MerchantCode: "YOUR_MERCHANT_CODE",
		APIKey:       "YOUR_API_KEY",
		IsSandbox:    true, // Set to false for production
	})

# Getting Available Payment Methods

Get available payment methods for a specific amount:

	paymentMethods, err := client.GetPaymentMethods(10000)
	if err != nil {
		log.Fatalf("Error getting payment methods: %v", err)
	}

	for _, method := range paymentMethods {
		fmt.Printf("Payment Method: %s (%s)\n", method.PaymentName, method.PaymentMethod)
	}

# Creating a Transaction

Create a transaction with the minimum required fields:

	transaction := duitku.TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   duitku.PaymentMethodBCA, // BCA Virtual Account
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		CustomerVaName:  "John Doe",
		Email:           "customer@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
	}

	result, err := client.CreateTransaction(transaction)
	if err != nil {
		log.Fatalf("Error creating transaction: %v", err)
	}

	fmt.Printf("Payment URL: %s\n", result.PaymentURL)
	fmt.Printf("Reference: %s\n", result.Reference)

# Validating a Transaction Locally

Validate is opt-in. CreateTransaction does not call it, so a request that
skips it is sent to Duitku and rejected there with HTTP 400. Calling Validate
first turns those rejections into a clearer local error:

	if err := transaction.Validate(); err != nil {
		log.Fatalf("Invalid transaction: %v", err)
	}

Validate enforces the documented rules: a minimum amount of 10000, the
documented field lengths, item totals that match paymentAmount, the per-channel
requirements, and the expiry window for the payment method.

# Creating a Transaction with Additional Details

Create a transaction with additional details like customer information and item
details:

	transaction := duitku.TransactionRequest{
		PaymentAmount:   40000,
		PaymentMethod:   duitku.PaymentMethodBCA,
		MerchantOrderID: "ORDER123",
		ProductDetails:  "Test Product",
		CustomerVaName:  "John Doe",
		Email:           "customer@example.com",
		PhoneNumber:     "08123456789",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",

		// Add customer details
		CustomerDetail: &duitku.CustomerDetail{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "customer@example.com",
			PhoneNumber: "08123456789",
			BillingAddress: &duitku.Address{
				FirstName:   "John",
				LastName:    "Doe",
				Address:     "Jl. Kembangan Raya",
				City:        "Jakarta",
				PostalCode:  "11530",
				Phone:       "08123456789",
				CountryCode: "ID",
			},
			ShippingAddress: &duitku.Address{
				FirstName:   "John",
				LastName:    "Doe",
				Address:     "Jl. Kembangan Raya",
				City:        "Jakarta",
				PostalCode:  "11530",
				Phone:       "08123456789",
				CountryCode: "ID",
			},
		},

		// Add item details. The total of price*quantity must equal paymentAmount.
		ItemDetails: []duitku.ItemDetail{
			{
				Name:     "Product 1",
				Price:    10000,
				Quantity: 1,
			},
			{
				Name:     "Product 2",
				Price:    30000,
				Quantity: 1,
			},
		},
	}

# Expiry Period

Leave ExpiryPeriod at zero to let Duitku apply the channel default, or look up
the documented window for a channel:

	if limits, ok := duitku.ExpiryPeriodFor(duitku.PaymentMethodQrisShopeePay); ok {
		fmt.Printf("Default %d minutes, maximum %d minutes\n", limits.Default, limits.Max)
	}

# Checking Transaction Status

Check the status of a transaction:

	status, err := client.CheckTransaction("ORDER123")
	if err != nil {
		log.Fatalf("Error checking transaction: %v", err)
	}

	switch status.StatusCode {
	case duitku.CheckTransactionStatusSuccess:
		fmt.Println("Paid")
	case duitku.CheckTransactionStatusPending:
		fmt.Println("Waiting for payment")
	case duitku.CheckTransactionStatusCancelled:
		fmt.Println("Canceled")
	}

A nil error only means the call succeeded. Check StatusCode before treating a
payment as complete. Duitku throttles this endpoint and can block a caller for
about an hour when the hit limit is reached, so do not poll it aggressively.

# Handling Callbacks

Handle callbacks from Duitku in your HTTP handler:

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		client.HandleCallback(w, r, func(data *duitku.CallbackData) error {
			// The signature has already been verified at this point.
			switch {
			case data.IsSuccessful():
				// Update order status in database
				fmt.Printf("Payment successful for order %s\n", data.MerchantOrderID)
			case data.IsFailed():
				fmt.Printf("Payment failed for order %s\n", data.MerchantOrderID)
			}
			return nil
		})
	})

HandleCallback answers HTTP 200 when the handler succeeds. Duitku retries a
callback up to five times when it does not receive a 200, so a handler error
answers 500 on purpose to trigger a retry.

Note that resultCode is not part of the signed string, so a captured callback
can have its resultCode changed without breaking the signature. Confirm the
final state with CheckTransaction before releasing goods or credit.

# Payment Methods

The package provides constants for every payment method code Duitku documents:

	// Credit Card
	duitku.PaymentMethodCreditCard // Credit Card (VC)

	// Virtual Account
	duitku.PaymentMethodBCA         // BCA Virtual Account (BC)
	duitku.PaymentMethodMandiri     // Mandiri Virtual Account (M2)
	duitku.PaymentMethodMaybank     // Maybank Virtual Account (VA)
	duitku.PaymentMethodBNI         // BNI Virtual Account (I1)
	duitku.PaymentMethodCIMB        // CIMB Niaga Virtual Account (B1)
	duitku.PaymentMethodPermata     // Permata Bank Virtual Account (BT)
	duitku.PaymentMethodATMBersama  // ATM Bersama (A1)
	duitku.PaymentMethodArthaGraha  // Bank Artha Graha Virtual Account (AG)
	duitku.PaymentMethodNeoCommerce // Bank Neo Commerce Virtual Account (NC)
	duitku.PaymentMethodBRI         // BRIVA / Bank BRI Virtual Account (BR)
	duitku.PaymentMethodSahabat     // Bank Sahabat Sampoerna Virtual Account (S1)
	duitku.PaymentMethodDanamon     // Danamon Virtual Account (DM)
	duitku.PaymentMethodBSI         // BSI Virtual Account (BV)

	// Retail Outlets
	duitku.PaymentMethodAlfamart  // Pegadaian/ALFA/Pos (FT)
	duitku.PaymentMethodPegadaian // Pegadaian/ALFA/Pos (FT), alias of Alfamart
	duitku.PaymentMethodPOS       // Pegadaian/ALFA/Pos (FT), alias of Alfamart
	duitku.PaymentMethodIndomaret // Indomaret (IR)

	// E-Wallet
	duitku.PaymentMethodOVO            // OVO (OV)
	duitku.PaymentMethodShopeePay      // Shopee Pay Apps (SA)
	duitku.PaymentMethodLinkAjaFixed   // LinkAja Apps Fixed Fee (LF)
	duitku.PaymentMethodLinkAjaPercent // LinkAja Apps Percentage Fee (LA)
	duitku.PaymentMethodDANA           // DANA (DA)
	duitku.PaymentMethodShopeeLink     // Shopee Pay Account Link (SL)
	duitku.PaymentMethodOVOLink        // OVO Account Link (OL)

	// E-Banking
	duitku.PaymentMethodJeniusPay // Jenius Pay (JP)

	// QRIS
	duitku.PaymentMethodQrisShopeePay // QRIS Shopee Pay (SP)
	duitku.PaymentMethodQrisNobu      // QRIS Nobu (NQ)
	duitku.PaymentMethodQrisNusapay   // QRIS Nusapay (SQ)

	// Paylater / credit. customerDetail and itemDetails are required.
	duitku.PaymentMethodIndodanaPaylater // Indodana Paylater (DN)
	duitku.PaymentMethodAtome            // ATOME (AT)

	// E-Commerce. customerVaName is required.
	duitku.PaymentMethodTokopediaCard    // Tokopedia Card Payment (T1)
	duitku.PaymentMethodTokopediaEWallet // Tokopedia E-Wallet (T2)
	duitku.PaymentMethodTokopediaOthers  // Tokopedia Others (T3)

# Transaction Status Codes

The package provides constants for each status code set. The callback, the
redirect and the check transaction endpoint use different meanings for the same
numbers, so they are kept apart:

	// inquiry response statusCode
	duitku.StatusSuccess // Success (00)
	duitku.StatusPending // Pending (01)

	// callback resultCode
	duitku.CallbackStatusSuccess // Success (00)
	duitku.CallbackStatusFailed  // Failed (01)

	// redirect resultCode appended to returnUrl
	duitku.RedirectStatusSuccess  // Success (00)
	duitku.RedirectStatusPending  // Pending (01)
	duitku.RedirectStatusCanceled // Canceled (02)

	// check transaction response statusCode
	duitku.CheckTransactionStatusSuccess   // Success (00)
	duitku.CheckTransactionStatusPending   // Pending (01)
	duitku.CheckTransactionStatusCancelled // Cancelled (02)

# Signatures

Every request is signed with HMAC-SHA256 and the callback signature is verified
the same way. The API key is the HMAC key, not a suffix of the message:

	get payment method: HMAC_SHA256(merchantcode + amount + datetime, apiKey)
	inquiry:            HMAC_SHA256(merchantCode + merchantOrderId + paymentAmount, apiKey)
	check transaction:  HMAC_SHA256(merchantCode + merchantOrderId, apiKey)
	callback:           HMAC_SHA256(merchantCode + amount + merchantOrderId, apiKey)
*/
package duitku
