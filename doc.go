/*
Package duitku provides a Go client for the Duitku Payment Gateway API.

Duitku is an Indonesian payment gateway that supports various payment methods
including virtual accounts, e-wallets, retail outlets, credit cards, and more.

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
		ExpiryPeriod:    10, // 10 minutes
	}

	result, err := client.CreateTransaction(transaction)
	if err != nil {
		log.Fatalf("Error creating transaction: %v", err)
	}

	fmt.Printf("Payment URL: %s\n", result.PaymentURL)
	fmt.Printf("Reference: %s\n", result.Reference)

# Creating a Transaction with Additional Details

Create a transaction with additional details like customer information and item details:

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
		ExpiryPeriod:    10,
		
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
		
		// Add item details
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

# Creating a Subscription Transaction

Create a recurring subscription transaction (only supported for credit card payments):

	isSubscription := true
	transaction := duitku.TransactionRequest{
		// Basic transaction details
		PaymentAmount:   50000,
		PaymentMethod:   duitku.PaymentMethodCreditCard,
		MerchantOrderID: "SUB123",
		ProductDetails:  "Monthly Subscription",
		CustomerVaName:  "John Doe",
		Email:           "customer@example.com",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    60,
		
		// Enable subscription
		IsSubscription: &isSubscription,
		
		// Subscription details
		SubscriptionDetail: &duitku.SubscriptionDetail{
			Description:      "Monthly Premium Plan",
			FrequencyType:    duitku.FrequencyMonthly,
			FrequencyInterval: 1,
			TotalNoOfCycles:   12,
		},
	}

# Checking Transaction Status

Check the status of a transaction:

	status, err := client.CheckTransaction("ORDER123")
	if err != nil {
		log.Fatalf("Error checking transaction: %v", err)
	}

	fmt.Printf("Status: %s (%s)\n", status.StatusMessage, status.StatusCode)

# Voiding a Transaction

Void a transaction (only supported for certain payment methods like OVO, ShopeePay, Credit Card):

	status, err := client.VoidTransaction("ORDER123", "REFERENCE123")
	if err != nil {
		log.Fatalf("Error voiding transaction: %v", err)
	}

	fmt.Printf("Void Status: %s (%s)\n", status.StatusMessage, status.StatusCode)

# Handling Callbacks

Handle callbacks from Duitku in your HTTP handler:

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		client.HandleCallback(w, r, func(data *duitku.CallbackData) error {
			// Check if payment is successful
			if data.IsSuccessful() {
				// Update order status in database
				fmt.Printf("Payment successful for order %s\n", data.MerchantOrderID)
			} else {
				// Handle failed payment
				fmt.Printf("Payment failed for order %s with code %s\n", 
					data.MerchantOrderID, data.ResultCode)
			}
			return nil
		})
	})

# Payment Methods

The package provides constants for all payment methods supported by Duitku:

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
	duitku.PaymentMethodBRI         // Bank BRI Virtual Account (BR)
	duitku.PaymentMethodSahabat     // Bank Sahabat Sampoerna Virtual Account (S1)
	duitku.PaymentMethodDanamon     // Danamon Virtual Account (DM)
	duitku.PaymentMethodBSI         // BSI Virtual Account (BV)

	// Retail Outlets
	duitku.PaymentMethodAlfamart  // Alfamart/Pegadaian/POS (FT)
	duitku.PaymentMethodPegadaian // Pegadaian (FT)
	duitku.PaymentMethodPOS       // POS (FT)
	duitku.PaymentMethodIndomaret // Indomaret (IR)

	// E-Wallet
	duitku.PaymentMethodOVO            // OVO (OV)
	duitku.PaymentMethodShopeePay      // Shopee Pay Apps (SA)
	duitku.PaymentMethodLinkAjaFixed   // LinkAja Apps Fixed Fee (LF)
	duitku.PaymentMethodLinkAjaPercent // LinkAja Apps Percentage Fee (LA)
	duitku.PaymentMethodDANA           // DANA (DA)
	duitku.PaymentMethodShopeeLink     // Shopee Pay Account Link (SL)
	duitku.PaymentMethodOVOLink        // OVO Account Link (OL)
	duitku.PaymentMethodJeniusPay      // Jenius Pay (JP)

	// QRIS
	duitku.PaymentMethodQrisShopeePay     // QRIS ShopeePay (SP)
	duitku.PaymentMethodQrisNobu          // QRIS Nobu (QN)
	duitku.PaymentMethodQrisDana          // QRIS Dana (DQ)
	duitku.PaymentMethodQrisGudangVoucher // QRIS Gudang Voucher (GQ)
	duitku.PaymentMethodQrisNusapay       // QRIS Nusapay (SQ)

	// Paylater/Credit
	duitku.PaymentMethodIndodanaPaylater // Indodana Paylater (ID)
	duitku.PaymentMethodAtome            // ATOME (AT)

# Transaction Status Codes

The package provides constants for transaction status codes:

	duitku.StatusSuccess    // Success (00)
	duitku.StatusPending    // Pending (01)

# Callback Status Codes

The package provides constants for callback status codes:

	duitku.CallbackStatusSuccess // Success (00)
	duitku.CallbackStatusFailed  // Failed (01)

# Check Transaction Status Codes

The package provides constants for check transaction status codes:

	duitku.CheckTransactionStatusSuccess   // Success (00)
	duitku.CheckTransactionStatusPending   // Pending (01)
	duitku.CheckTransactionStatusCancelled // Cancelled (02)

# Subscription Frequency Types

For credit card subscription transactions, the package provides constants for frequency types:

	duitku.FrequencyDaily   // Daily (1)
	duitku.FrequencyWeekly  // Weekly (2)
	duitku.FrequencyMonthly // Monthly (3)
	duitku.FrequencyYearly  // Yearly (4)
*/
package duitku
