# Duitku Go 💸

[![Go](https://github.com/fatkulnurk/duitku-go/actions/workflows/go.yml/badge.svg)](https://github.com/fatkulnurk/duitku-go/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/fatkulnurk/duitku-go/branch/main/graph/badge.svg)](https://codecov.io/gh/fatkulnurk/duitku-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/fatkulnurk/duitku-go)](https://goreportcard.com/report/github.com/fatkulnurk/duitku-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/fatkulnurk/duitku-go.svg)](https://pkg.go.dev/github.com/fatkulnurk/duitku-go)

A simple and lightweight Duitku.com Payment Gateway SDK for Golang — built with only Go's standard library. No external dependencies, making it ideal for minimal and secure payment gateway integrations. This package implements [Duitku API v2](https://docs.duitku.com/api/id/#langkah-awal).

## Table of Contents
- [🌟 Overview](#-overview)
- [📋 Requirements](#-requirements)
- [📦 Installation](#-installation)
- [📚 Usage](#-usage)
- [✨ Features](#-features)
  - [Payment Methods 💳](#payment-methods-)
  - [Transactions 🛒](#transactions-)
  - [Callbacks 📡](#callbacks-)
  - [Validation 🔍](#validation-)
  - [Additional Features 🔧](#additional-features-)
  - [Payment Methods Supported 💰](#payment-methods-supported-)
- [🔐 Signatures](#-signatures)
- [⚠️ Migrating from v0.x](#️-migrating-from-v0x)
- [🧪 Example Application](#-example-application)
- [👥 Contributing](#-contributing)
- [📄 License](#-license)

## 🌟 Overview

Duitku is a payment gateway service that provides various payment methods for Indonesian merchants. This Go package provides a clean, idiomatic interface to integrate with Duitku's payment services.

[🔼 Jump to Table of Contents](#table-of-contents)

## 📋 Requirements

- Go 1.22 or higher
- Duitku merchant account and API credentials

[🔼 Jump to Table of Contents](#table-of-contents)

## 📦 Installation

```bash
go get github.com/fatkulnurk/duitku-go
```

[🔼 Jump to Table of Contents](#table-of-contents)

## 📚 Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/fatkulnurk/duitku-go"
)

func main() {
	// Initialize the client
	client := duitku.NewClient(duitku.Config{
		MerchantCode:               "YOUR_MERCHANT_CODE",
		APIKey:                     "YOUR_API_KEY",
		IsSandbox:                  true, // Set to false for production
		LogEveryRequestAndResponse: true, // Set to false to disable logging every request and response
	})

	// Get available payment methods
	paymentMethods, err := client.GetPaymentMethods(10000)
	if err != nil {
		log.Fatalf("Error getting payment methods: %v", err)
	}

	for _, method := range paymentMethods {
		fmt.Printf("Payment Method: %s (%s)\n", method.PaymentName, method.PaymentMethod)
	}

	// Create a transaction
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

	if err := transaction.Validate(); err != nil {
		log.Fatalf("Invalid transaction: %v", err)
	}

	result, err := client.CreateTransaction(transaction)
	if err != nil {
		log.Fatalf("Error creating transaction: %v", err)
	}

	fmt.Printf("Payment URL: %s\n", result.PaymentURL)
	fmt.Printf("Reference: %s\n", result.Reference)

	// Check transaction status
	status, err := client.CheckTransaction("ORDER123")
	if err != nil {
		log.Fatalf("Error checking transaction: %v", err)
	}

	fmt.Printf("Status: %s (%s)\n", status.StatusMessage, status.StatusCode)
}
```

[🔼 Jump to Table of Contents](#table-of-contents)

## ✨ Features

### Payment Methods 💳

- ✅ **Get Available Payment Methods** - [API Reference](https://docs.duitku.com/api/id/#get-payment-method)
  - Retrieve all available payment methods based on transaction amount
  - Returns the methods, fees and images Duitku has active for your project

[🔼 Jump to Table of Contents](#table-of-contents)

### Transactions 🛒

- ✅ **Create Transaction** - [API Reference](https://docs.duitku.com/api/id/#permintaan-transaksi)
  - Create payment invoices with complete customer details
  - Support for item details, account link and credit card details
  - Configurable expiry period, or omit it to use the channel default
  - Returns payment URL, VA number, QR string and reference

- ✅ **Check Transaction Status** - [API Reference](https://docs.duitku.com/api/id/#cek-transaksi)
  - Verify payment status using merchant order ID
  - Get detailed transaction information including amount and fee
  - Distinguishes success, pending and canceled through the status constants

[🔼 Jump to Table of Contents](#table-of-contents)

### Callbacks 📡

- ✅ **Handle Payment Notifications** - [API Reference](https://docs.duitku.com/api/id/#callback)
  - Secure callback handling with HMAC-SHA256 signature verification
  - Constant-time signature comparison
  - Rejects callbacks addressed to a different merchant code
  - Answers HTTP 200 on success and 500 on a handler failure so Duitku retries

[🔼 Jump to Table of Contents](#table-of-contents)

### Validation 🔍

`TransactionRequest.Validate` is opt-in and enforces the rules Duitku documents,
so a request fails locally with a clear message instead of being rejected by
Duitku with an opaque HTTP 400:

- Minimum payment amount of 10000
- Documented field lengths for `merchantOrderId` and `productDetails`
- Item totals: the sum of `price * quantity` must equal `paymentAmount`
- Per-channel requirements: `customerDetail` and `itemDetails` for paylater,
  `customerVaName` for e-commerce
- Expiry windows per payment method, including channels that ignore the
  requested value entirely

[🔼 Jump to Table of Contents](#table-of-contents)

### Additional Features 🔧

- ✅ **Environment Support**
  - Sandbox environment for testing - [Sandbox Dashboard](https://sandbox.duitku.com/)
  - Production environment for live transactions

- ✅ **Security**
  - HMAC-SHA256 signature generation and constant-time verification
  - Secure API key handling
  - Rejects requests when `MerchantCode` or `APIKey` is empty

- ✅ **Robust Error Reporting**
  - Typed `APIError` carrying the HTTP status, Duitku code and message
  - Reads the documented `Message` error key
  - A non-JSON or empty error body is reported as an HTTP failure, never as a
    decoding error

[🔼 Jump to Table of Contents](#table-of-contents)

### Payment Methods Supported 💰

| Category | Payment Method | Code |
|----------|---------------|------|
| **Credit Card** | Credit Card | VC |
| **Virtual Account** | BCA VA | BC |
| | Mandiri VA | M2 |
| | Maybank VA | VA |
| | BNI VA | I1 |
| | CIMB Niaga VA | B1 |
| | Permata VA | BT |
| | ATM Bersama | A1 |
| | Bank Artha Graha | AG |
| | Bank Neo Commerce/BNC | NC |
| | BRIVA | BR |
| | Bank Sahabat Sampoerna | S1 |
| | Danamon VA | DM |
| | BSI VA | BV |
| **Retail Outlets** | Pegadaian/ALFA/Pos | FT |
| | Indomaret | IR |
| **E-Wallet** | OVO | OV |
| | Shopee Pay Apps | SA |
| | LinkAja (Fixed Fee) | LF |
| | LinkAja (Percentage Fee) | LA |
| | DANA | DA |
| | Shopee Pay Account Link | SL |
| | OVO Account Link | OL |
| **E-Banking** | Jenius Pay | JP |
| **QRIS** | QRIS Shopee Pay | SP |
| | QRIS Nobu | NQ |
| | QRIS Nusapay | SQ |
| **Paylater** | Indodana Paylater | DN |
| | Atome | AT |
| **E-Commerce** | Tokopedia Card Payment | T1 |
| | Tokopedia E-Wallet | T2 |
| | Tokopedia Others | T3 |

[🔼 Jump to Table of Contents](#table-of-contents)

## 🔐 Signatures

Duitku API v2 signs every request with HMAC-SHA256, using your API key as the
HMAC key. The message is the concatenated parameters and the output is lowercase
hex:

| Operation | stringToSign |
|---|---|
| Get Payment Method | `merchantcode + amount + datetime` |
| Create Transaction | `merchantCode + merchantOrderId + paymentAmount` |
| Check Transaction | `merchantCode + merchantOrderId` |
| Callback | `merchantCode + amount + merchantOrderId` |

The older MD5 and plain SHA256 schemes are obsolete and are not implemented.
This package handles signing and callback verification for you; you never build
a signature yourself.

[🔼 Jump to Table of Contents](#table-of-contents)

## ⚠️ Migrating from v0.x

This release aligns the SDK with the current Duitku API v2 documentation. If you
are upgrading from v0.x, note the following breaking changes:

- **Signatures are now HMAC-SHA256.** v0.x computed MD5 and plain SHA256
  digests with the API key appended to the message. Those schemes are obsolete
  and are gone. No code change is needed on your side.
- **`VoidTransaction` was removed.** The endpoint it called is not part of the
  documented API v2.
- **`IsSubscription` and `SubscriptionDetail` were removed.** Duitku removed
  the subscription feature from the inquiry request in November 2025.
- **Corrected payment method codes.** `PaymentMethodQrisNobu` is now `NQ` (was
  `QN`) and `PaymentMethodIndodanaPaylater` is now `DN` (was `ID`). If you
  stored the old values, update them.
- **Removed stale payment methods.** `PaymentMethodQrisDana` and
  `PaymentMethodQrisGudangVoucher` were removed because Duitku retired those
  channels. Tokopedia constants `T1`, `T2` and `T3` were added.
- **`TransactionResponse.Amount` and `TransactionStatusResponse.Amount` now use
  `StringOrInt`** so they decode whether Duitku sends `"40000"` or `40000`. Use
  `.String()` or `.Int()` instead of comparing to a string literal.
- **`ExpiryPeriod` is omitted when zero** instead of being sent as `0`.

[🔼 Jump to Table of Contents](#table-of-contents)

## 🧪 Example Application

This package includes a fully functional example application that demonstrates how to use all the features of the Duitku Go client. The example application is a simple web server that allows you to:

- Create payment transactions
- Handle payment callbacks
- Check transaction status
- View payment details

To run the example application:

```bash
# Set your Duitku credentials
export DUITKU_MERCHANT_CODE="your_merchant_code"
export DUITKU_API_KEY="your_api_key"

# Run the example application
cd example
go run main.go
```

Then open your browser to http://localhost:8080 to see the example application in action.

[🔼 Jump to Table of Contents](#table-of-contents)

## 👥 Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests to ensure everything works (`go test ./...`)
4. Commit your changes (`git commit -m 'Add some amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

Please make sure your code passes all tests and only use Go's standard library.

[🔼 Jump to Table of Contents](#table-of-contents)

## 📄 License

MIT License - See [LICENSE](LICENSE) for details.
