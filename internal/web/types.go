package web

import "net/http"

type Handlers interface {
	CheckoutHandler(w http.ResponseWriter, r *http.Request)
	CheckoutInitiateAuthHandler(w http.ResponseWriter, r *http.Request)
	CheckoutProcessAuthHandler(w http.ResponseWriter, r *http.Request)
	CheckoutPayHandler(w http.ResponseWriter, r *http.Request)
}
