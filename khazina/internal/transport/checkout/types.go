package checkout

import "net/http"

type Handlers interface {
	CheckoutPageHandler(w http.ResponseWriter, r *http.Request)
	CreatePaymentIntentHandler(w http.ResponseWriter, r *http.Request)
	PrepareCardAuthenticationHandler(w http.ResponseWriter, r *http.Request)
	AuthenticateCardholderHandler(w http.ResponseWriter, r *http.Request)
	CardAuthenticationReturnHandler(w http.ResponseWriter, r *http.Request)
	VerifyCardAuthenticationHandler(w http.ResponseWriter, r *http.Request)
	CapturePaymentIntentHandler(w http.ResponseWriter, r *http.Request)
	VerifyDomainHandler(w http.ResponseWriter, r *http.Request)
}
