package duitku

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// SandboxBaseURL is the base URL for the Duitku sandbox environment
	SandboxBaseURL = "https://sandbox.duitku.com/webapi/api"
	// ProductionBaseURL is the base URL for the Duitku production environment
	ProductionBaseURL = "https://passport.duitku.com/webapi/api"
)

// maxLoggedBodyLength caps how much of a raw response body is embedded in an
// error message, so an unexpected HTML error page cannot flood logs.
const maxLoggedBodyLength = 256

// duitkuJSONOptions are the JSON options applied to every Duitku payload.
//
// Duitku is a third party gateway whose own documentation disagrees about key
// casing: the transaction response example spells the e-commerce link "AppUrl"
// while the parameter table spells it "appUrl". Failing an otherwise valid
// payment response over a casing difference is worse than decoding it, so
// case-insensitive name matching is enabled explicitly. encoding/json/v2 is
// case sensitive by default; the v1 package this code previously relied on was
// not.
//
// AllowDuplicateNames is enabled for the same reason. v2 rejects duplicate
// object members by default while v1 permitted them, and a repeated key must
// not turn a payment response into a decoding error.
var duitkuJSONOptions = []jsonv2.Options{
	jsonv2.MatchCaseInsensitiveNames(true),
	jsontext.AllowDuplicateNames(true),
}

// Config holds the configuration for the Duitku client
type Config struct {
	// MerchantCode is the merchant code provided by Duitku
	MerchantCode string
	// APIKey is the API key provided by Duitku
	APIKey string
	// IsSandbox determines whether to use the sandbox or production environment
	IsSandbox bool
	// HTTPClient is an optional custom HTTP client
	HTTPClient *http.Client
	// Logger is an optional custom logger
	Logger *log.Logger
	// LogEveryRequestAndResponse logs every request and response
	LogEveryRequestAndResponse bool
}

// Client is the Duitku API client
type Client struct {
	config                     Config
	baseURL                    string
	httpClient                 *http.Client
	logger                     *log.Logger
	logEveryRequestAndResponse bool
}

// NewClient creates a new Duitku client with the provided configuration
func NewClient(config Config) *Client {
	baseURL := ProductionBaseURL
	if config.IsSandbox {
		baseURL = SandboxBaseURL
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	} else if httpClient.Timeout == 0 {
		// A caller-supplied client without a timeout would block forever.
		// Copy it so we do not mutate the caller's value.
		cloned := *httpClient
		cloned.Timeout = 30 * time.Second
		httpClient = &cloned
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "github.com/fatkulnurk/duitku-go: ", log.LstdFlags)
	}

	return &Client{
		config:                     config,
		baseURL:                    baseURL,
		httpClient:                 httpClient,
		logger:                     logger,
		logEveryRequestAndResponse: config.LogEveryRequestAndResponse,
	}
}

// validateCredentials reports whether the client has the credentials required
// to sign a request. Signing with an empty API key would produce a signature
// that anyone could reproduce, so this is checked before every call.
func (c *Client) validateCredentials() error {
	if c.config.MerchantCode == "" {
		return errors.New("duitku: MerchantCode is required")
	}
	if c.config.APIKey == "" {
		return errors.New("duitku: APIKey is required")
	}
	return nil
}

// createSignatureHMACSHA256 builds the Duitku API v2 signature.
//
//	stringToSign = the parameters concatenated in the order given
//	signature    = HMAC_SHA256(stringToSign, apiKey), lowercase hex
//
// Every Duitku v2 operation uses this construction. The older MD5 and plain
// SHA256 schemes are obsolete:
// https://docs.duitku.com/api/id/#permintaan-transaksi
func (c *Client) createSignatureHMACSHA256(params ...string) string {
	var stringToSign strings.Builder
	for _, param := range params {
		stringToSign.WriteString(param)
	}

	mac := hmac.New(sha256.New, []byte(c.config.APIKey))
	mac.Write([]byte(stringToSign.String()))

	return hex.EncodeToString(mac.Sum(nil))
}

// doRequest performs an HTTP request to the Duitku API
func (c *Client) doRequest(method, endpoint string, body interface{}, result interface{}) error {
	if err := c.validateCredentials(); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := jsonv2.Marshal(body, duitkuJSONOptions...)
		if err != nil {
			return fmt.Errorf("duitku: marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("duitku: creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("duitku: making request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("duitku: reading response body (HTTP %d): %w", resp.StatusCode, err)
	}

	if c.logEveryRequestAndResponse {
		c.logger.Println("-=-=-=-= [client.go][doRequest] -=-=-=-=")
		c.logger.Printf("Request method: %s \n", method)
		c.logger.Printf("Request url: %s \n", url)
		c.logger.Printf("Request body: %+v \n", body)
		c.logger.Printf("Response status code: %d \n", resp.StatusCode)
		c.logger.Printf("Response status: %s \n", resp.Status)
		c.logger.Printf("Response body: %s\n", string(rawBody))
		c.logger.Println("-=-=-=-= [client.go][doRequest] -=-=-=-=")
	}

	if resp.StatusCode != http.StatusOK {
		return newAPIError(resp.StatusCode, rawBody)
	}

	if result == nil {
		return nil
	}

	if err := jsonv2.Unmarshal(rawBody, result, duitkuJSONOptions...); err != nil {
		return fmt.Errorf("duitku: decoding response (HTTP %d): %w", resp.StatusCode, err)
	}

	return nil
}

// APIError is returned when Duitku answers with a non-200 HTTP status.
//
// Duitku documents HTTP 400/401/404/409 with a body carrying a "Message" key
// (capital M), which differs from the success envelopes that use
// responseCode/responseMessage or statusCode/statusMessage. All of those
// spellings are accepted here so the real reason is never lost.
// https://docs.duitku.com/api/id/#http-code
type APIError struct {
	// StatusCode is the HTTP status code returned by Duitku
	StatusCode int
	// Code is the Duitku error code when one is present
	Code string
	// Message is the human readable reason reported by Duitku
	Message string
	// RawBody is the raw response body, truncated for safety
	RawBody string
}

// Error returns the error message
func (e *APIError) Error() string {
	message := e.Message
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}

	if e.Code != "" {
		return fmt.Sprintf("duitku: API error (HTTP %d, code %s): %s", e.StatusCode, e.Code, message)
	}

	return fmt.Sprintf("duitku: API error (HTTP %d): %s", e.StatusCode, message)
}

// errorEnvelope covers every error body shape Duitku is known to send.
type errorEnvelope struct {
	Message         string `json:"Message"`
	ResponseMessage string `json:"responseMessage"`
	StatusMessage   string `json:"statusMessage"`
	ResponseCode    string `json:"responseCode"`
	StatusCode      string `json:"statusCode"`
}

// newAPIError builds an APIError from a non-200 response. A body that is not
// JSON (an HTML gateway page, an empty body, a rate-limit block) must not be
// reported as a decoding failure, so parsing is best effort.
func newAPIError(statusCode int, rawBody []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		RawBody:    truncate(strings.TrimSpace(string(rawBody)), maxLoggedBodyLength),
	}

	var envelope errorEnvelope
	if err := jsonv2.Unmarshal(rawBody, &envelope, duitkuJSONOptions...); err == nil {
		switch {
		case envelope.Message != "":
			apiErr.Message = envelope.Message
		case envelope.ResponseMessage != "":
			apiErr.Message = envelope.ResponseMessage
		case envelope.StatusMessage != "":
			apiErr.Message = envelope.StatusMessage
		}

		switch {
		case envelope.ResponseCode != "":
			apiErr.Code = envelope.ResponseCode
		case envelope.StatusCode != "":
			apiErr.Code = envelope.StatusCode
		}
	}

	if apiErr.Message == "" {
		if apiErr.RawBody != "" {
			apiErr.Message = apiErr.RawBody
		} else {
			apiErr.Message = http.StatusText(statusCode)
		}
	}

	return apiErr
}

// truncate shortens s to at most max runes, appending an ellipsis when cut.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
