package checkout

import "net/http"

type Handlers interface {
	CheckoutPageHandler(w http.ResponseWriter, r *http.Request)
	InitiateSessionHandler(w http.ResponseWriter, r *http.Request)
	CardInitiateAuthHandler(w http.ResponseWriter, r *http.Request)
	CardProcessAuthHandler(w http.ResponseWriter, r *http.Request)
	CardFinalizeHandler(w http.ResponseWriter, r *http.Request)
	WalletPayHandler(w http.ResponseWriter, r *http.Request)
	VerifyDomainHandler(w http.ResponseWriter, r *http.Request)
}
