package duitku

// Payment method codes as published in the Duitku payment method table.
// https://docs.duitku.com/api/id/#metode-pembayaran
const (
	// Credit Card
	PaymentMethodCreditCard = "VC" // Credit Card (Visa / Master Card / JCB)

	// Virtual Account
	PaymentMethodBCA         = "BC" // BCA Virtual Account
	PaymentMethodMandiri     = "M2" // Mandiri Virtual Account
	PaymentMethodMaybank     = "VA" // Maybank Virtual Account
	PaymentMethodBNI         = "I1" // BNI Virtual Account
	PaymentMethodCIMB        = "B1" // CIMB Niaga Virtual Account
	PaymentMethodPermata     = "BT" // Permata Bank Virtual Account
	PaymentMethodATMBersama  = "A1" // ATM Bersama
	PaymentMethodArthaGraha  = "AG" // Bank Artha Graha Virtual Account
	PaymentMethodNeoCommerce = "NC" // Bank Neo Commerce / BNC Virtual Account
	PaymentMethodBRI         = "BR" // BRIVA / Bank BRI Virtual Account
	PaymentMethodSahabat     = "S1" // Bank Sahabat Sampoerna Virtual Account
	PaymentMethodDanamon     = "DM" // Danamon Virtual Account
	PaymentMethodBSI         = "BV" // BSI Virtual Account
)

// Retail outlets.
//
// Duitku exposes a single code, FT, for the combined "Pegadaian/ALFA/Pos"
// channel, so the three constants below are aliases of one another. Duitku
// cannot tell them apart.
const (
	PaymentMethodAlfamart  = "FT" // Pegadaian / ALFA / Pos
	PaymentMethodPegadaian = "FT" // Pegadaian / ALFA / Pos
	PaymentMethodPOS       = "FT" // Pegadaian / ALFA / Pos
	PaymentMethodIndomaret = "IR" // Indomaret
)

// E-Wallet
const (
	PaymentMethodOVO            = "OV" // OVO
	PaymentMethodShopeePay      = "SA" // Shopee Pay Apps
	PaymentMethodLinkAjaFixed   = "LF" // LinkAja Apps (Fixed Fee)
	PaymentMethodLinkAjaPercent = "LA" // LinkAja Apps (Percentage Fee)
	PaymentMethodDANA           = "DA" // DANA
	PaymentMethodShopeeLink     = "SL" // Shopee Pay Account Link
	PaymentMethodOVOLink        = "OL" // OVO Account Link
)

// E-Banking
const (
	PaymentMethodJeniusPay = "JP" // Jenius Pay
)

// QRIS
const (
	PaymentMethodQrisShopeePay = "SP" // QRIS Shopee Pay
	PaymentMethodQrisNobu      = "NQ" // QRIS Nobu
	PaymentMethodQrisNusapay   = "SQ" // QRIS Nusapay
)

// Paylater / credit. These channels require both customerDetail and
// itemDetails to be present.
const (
	PaymentMethodIndodanaPaylater = "DN" // Indodana Paylater
	PaymentMethodAtome            = "AT" // ATOME
)

// E-Commerce. These channels require customerVaName to be present.
const (
	PaymentMethodTokopediaCard    = "T1" // Tokopedia Card Payment
	PaymentMethodTokopediaEWallet = "T2" // Tokopedia E-Wallet
	PaymentMethodTokopediaOthers  = "T3" // Tokopedia Others
)

// Transaction status codes returned in statusCode by the inquiry endpoint.
const (
	StatusSuccess = "00" // Success
	StatusPending = "01" // Pending
)

// Callback resultCode values. The callback is the authoritative payment
// notification.
const (
	CallbackStatusSuccess = "00" // Success
	CallbackStatusFailed  = "01" // Failed
)

// Redirect resultCode values appended to returnUrl. These differ from the
// callback codes: on redirect, 01 means pending, not failed.
//
// Duitku warns that returnUrl is user controlled, so never use it to change a
// payment status.
const (
	RedirectStatusSuccess  = "00" // Success
	RedirectStatusPending  = "01" // Pending
	RedirectStatusCanceled = "02" // Canceled
)

// Check Transaction statusCode values returned by the transaction status
// endpoint.
const (
	CheckTransactionStatusSuccess   = "00" // Success
	CheckTransactionStatusPending   = "01" // Pending
	CheckTransactionStatusCancelled = "02" // Canceled
)

// MinimumPaymentAmount is the smallest payment Duitku accepts. A smaller amount
// is rejected with HTTP 400 "Minimum Payment 10000 IDR".
const MinimumPaymentAmount = 10000

// MaximumMerchantOrderIDLength is the documented limit for merchantOrderId.
const MaximumMerchantOrderIDLength = 50

// MaximumProductDetailsLength is the documented limit for productDetails.
const MaximumProductDetailsLength = 255
