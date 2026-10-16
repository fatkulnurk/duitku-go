package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fatkulnurk/duitku-go"
)

func main() {
	// Get merchant code and API key from environment variables
	merchantCode := os.Getenv("DUITKU_MERCHANT_CODE")
	apiKey := os.Getenv("DUITKU_API_KEY")

	if merchantCode == "" || apiKey == "" {
		log.Fatal("DUITKU_MERCHANT_CODE and DUITKU_API_KEY environment variables must be set")
	}

	// Initialize Duitku client
	client := duitku.NewClient(duitku.Config{
		MerchantCode:               merchantCode,
		APIKey:                     apiKey,
		IsSandbox:                  true, // Use sandbox environment for testing
		LogEveryRequestAndResponse: true,
	})

	// Set up HTTP server to handle routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/payment", func(w http.ResponseWriter, r *http.Request) {
		paymentHandler(w, r, client)
	})
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		callbackHandler(w, r, client)
	})
	http.HandleFunc("/return", returnHandler)
	http.HandleFunc("/check-status", func(w http.ResponseWriter, r *http.Request) {
		checkStatusHandler(w, r, client)
	})

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// homeHandler displays a simple payment form
func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Duitku Payment Example</title>
		<style>
			body { font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; }
			.form-group { margin-bottom: 15px; }
			label { display: block; margin-bottom: 5px; }
			input, select { width: 100%; padding: 8px; box-sizing: border-box; }
			button { background: #4CAF50; color: white; padding: 10px 15px; border: none; cursor: pointer; }
		</style>
	</head>
	<body>
		<h1>Duitku Payment Example</h1>
		<form action="/payment" method="post">
			<div class="form-group">
				<label for="amount">Amount (IDR)</label>
				<input type="number" id="amount" name="amount" value="10000" required>
			</div>
			<div class="form-group">
				<label for="payment_method">Payment Method</label>
				<select id="payment_method" name="payment_method" required>
					<!-- Credit Card -->
					<optgroup label="Credit Card">
						<option value="VC">Credit Card (Visa / Master Card / JCB)</option>
					</optgroup>
					
					<!-- Virtual Account -->
					<optgroup label="Virtual Account">
						<option value="BC">BCA Virtual Account</option>
						<option value="M2">Mandiri Virtual Account</option>
						<option value="VA">Maybank Virtual Account</option>
						<option value="I1">BNI Virtual Account</option>
						<option value="B1">CIMB Niaga Virtual Account</option>
						<option value="BT">Permata Bank Virtual Account</option>
						<option value="A1">ATM Bersama</option>
						<option value="AG">Bank Artha Graha</option>
						<option value="NC">Bank Neo Commerce/BNC</option>
						<option value="BR">BRIVA</option>
						<option value="S1">Bank Sahabat Sampoerna</option>
						<option value="DM">Danamon Virtual Account</option>
						<option value="BV">BSI Virtual Account</option>
					</optgroup>
					
					<!-- Ritel -->
					<optgroup label="Retail Outlets">
						<option value="FT">Pegadaian/ALFA/Pos</option>
						<option value="IR">Indomaret</option>
					</optgroup>
					
					<!-- E-Wallet -->
					<optgroup label="E-Wallet">
						<option value="OV">OVO</option>
						<option value="SA">Shopee Pay Apps</option>
						<option value="LF">LinkAja Apps (Fixed Fee)</option>
						<option value="LA">LinkAja Apps (Percentage Fee)</option>
						<option value="DA">DANA</option>
						<option value="SL">Shopee Pay Account Link</option>
						<option value="OL">OVO Account Link</option>
					</optgroup>
					
					<!-- E-Banking -->
					<optgroup label="E-Banking">
						<option value="JP">Jenius Pay</option>
					</optgroup>
					
					<!-- QRIS -->
					<optgroup label="QRIS">
						<option value="SP">QRIS ShopeePay</option>
						<option value="NQ">QRIS Nobu</option>
						<option value="SQ">QRIS Nusapay</option>
					</optgroup>
					
					<!-- Kredit -->
					<optgroup label="Paylater/Credit">
						<option value="DN">Indodana Paylater</option>
						<option value="AT">ATOME</option>
					</optgroup>
					
					<!-- E-Commerce -->
					<optgroup label="E-Commerce">
						<option value="T1">Tokopedia Card Payment</option>
						<option value="T2">Tokopedia E-Wallet</option>
						<option value="T3">Tokopedia Others</option>
					</optgroup>
				</select>
			</div>
			<div class="form-group">
				<label for="email">Email</label>
				<input type="email" id="email" name="email" value="customer@example.com" required>
			</div>
			<div class="form-group">
				<label for="name">Name</label>
				<input type="text" id="name" name="name" value="John Doe" required>
			</div>
			<button type="submit">Pay Now</button>
		</form>
	</body>
	</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// paymentHandler processes the payment form and creates a transaction
func paymentHandler(w http.ResponseWriter, r *http.Request, client *duitku.Client) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Get form values
	amountStr := r.FormValue("amount")
	paymentMethod := r.FormValue("payment_method")
	email := r.FormValue("email")
	name := r.FormValue("name")

	// Parse amount
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	// Generate unique merchant order ID
	merchantOrderID := fmt.Sprintf("ORDER-%d", time.Now().Unix())

	// Create transaction request
	transaction := duitku.TransactionRequest{
		PaymentAmount:   amount,
		PaymentMethod:   paymentMethod,
		MerchantOrderID: merchantOrderID,
		ProductDetails:  "Product from Duitku Go Example",
		CustomerVaName:  name,
		Email:           email,
		PhoneNumber:     "08123456789", // Example phone number
		CallbackURL:     fmt.Sprintf("%s/callback", getBaseURL(r)),
		ReturnURL:       fmt.Sprintf("%s/return?order_id=%s", getBaseURL(r), merchantOrderID),

		// Add customer details
		CustomerDetail: &duitku.CustomerDetail{
			FirstName:   name,
			Email:       email,
			PhoneNumber: "08123456789",
		},

		// Add item details
		ItemDetails: []duitku.ItemDetail{
			{
				Name:     "Example Product",
				Price:    amount,
				Quantity: 1,
			},
		},
	}

	// Leave expiryPeriod unset for channels that always use their own default.
	if limits, ok := duitku.ExpiryPeriodFor(paymentMethod); ok && !limits.Fixed {
		transaction.ExpiryPeriod = limits.Default
	}

	// Fail fast with a clear message before calling Duitku.
	if err := transaction.Validate(); err != nil {
		log.Printf("Invalid transaction: %v", err)
		http.Error(w, "Invalid transaction: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create transaction
	result, err := client.CreateTransaction(transaction)
	if err != nil {
		log.Printf("Error creating transaction: %v", err)
		http.Error(w, "Error creating transaction", http.StatusInternalServerError)
		return
	}

	// Store transaction details in database (in a real application)
	// ...

	// Redirect to payment URL
	http.Redirect(w, r, result.PaymentURL, http.StatusSeeOther)
}

// callbackHandler processes callbacks from Duitku
func callbackHandler(w http.ResponseWriter, r *http.Request, client *duitku.Client) {
	client.HandleCallback(w, r, func(data *duitku.CallbackData) error {
		// Log callback data
		log.Printf("Received callback: OrderID=%s, Amount=%s, Status=%s",
			data.MerchantOrderID, data.Amount, data.ResultCode)

		// Check if payment is successful
		switch {
		case data.IsSuccessful():
			// Update order status in database (in a real application)
			// ...
			log.Printf("Payment successful for order %s", data.MerchantOrderID)
		case data.IsFailed():
			log.Printf("Payment failed for order %s", data.MerchantOrderID)
		default:
			log.Printf("Unexpected resultCode %s for order %s", data.ResultCode, data.MerchantOrderID)
		}

		return nil
	})
}

// returnHandler handles the user return from the payment page
func returnHandler(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("order_id")

	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>Payment Return</title>
		<style>
			body { font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; }
			.card { border: 1px solid #ddd; border-radius: 4px; padding: 20px; margin-top: 20px; }
			.btn { display: inline-block; background: #4CAF50; color: white; padding: 10px 15px; text-decoration: none; border-radius: 4px; }
		</style>
	</head>
	<body>
		<h1>Thank You</h1>
		<p>Your payment for order %s is being processed.</p>
		<div class="card">
			<h2>Order Details</h2>
			<p>Order ID: %s</p>
			<p>You can check the status of your payment below:</p>
			<a href="/check-status?order_id=%s" class="btn">Check Payment Status</a>
		</div>
		<p><a href="/">Back to Home</a></p>
	</body>
	</html>
	`, orderID, orderID, orderID)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// checkStatusHandler checks the status of a transaction
func checkStatusHandler(w http.ResponseWriter, r *http.Request, client *duitku.Client) {
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	// Check transaction status
	status, err := client.CheckTransaction(orderID)
	if err != nil {
		log.Printf("Error checking transaction: %v", err)
		http.Error(w, "Error checking transaction status", http.StatusInternalServerError)
		return
	}

	// Display status
	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>Payment Status</title>
		<style>
			body { font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; }
			.card { border: 1px solid #ddd; border-radius: 4px; padding: 20px; margin-top: 20px; }
			.status { font-weight: bold; }
			.success { color: green; }
			.pending { color: orange; }
			.canceled { color: gray; }
			.failed { color: red; }
		</style>
	</head>
	<body>
		<h1>Payment Status</h1>
		<div class="card">
			<h2>Order Details</h2>
			<p>Order ID: %s</p>
			<p>Reference: %s</p>
			<p>Amount: Rp %s</p>
			<p>Status: <span class="status %s">%s (%s)</span></p>
		</div>
		<p><a href="/">Back to Home</a></p>
	</body>
	</html>
	`,
		status.MerchantOrderID,
		status.Reference,
		status.Amount,
		getStatusClass(status.StatusCode),
		status.StatusMessage,
		status.StatusCode)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// getBaseURL returns the base URL of the current request.
//
// It honours X-Forwarded-Proto so that a deployment behind a TLS terminating
// proxy does not advertise an http callback URL to Duitku.
func getBaseURL(r *http.Request) string {
	scheme := "http"

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// getStatusClass returns the CSS class for a status code
func getStatusClass(statusCode string) string {
	switch statusCode {
	case duitku.CheckTransactionStatusSuccess:
		return "success"
	case duitku.CheckTransactionStatusPending:
		return "pending"
	case duitku.CheckTransactionStatusCancelled:
		return "canceled"
	default:
		return "failed"
	}
}
