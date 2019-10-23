package svc

import (
	"net/http"
)

// HS500 -- returns 500 status code.
func HS500(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache,no-store")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 Internal Server Error"))
}

// HS400 -- returns 400 status code.
func HS400(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache,no-store")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 Bad Request"))
}

// HS400t -- returns 400 status code with an error message.
func HS400t(w http.ResponseWriter, errmsg string) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache,no-store")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 Bad Request: " + errmsg))
}

// HS200j -- returns 200 status code and writes `b` as JSON.
func HS200j(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache,no-store")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// HS200t -- returns 200 status code and writes `b` as text.
func HS200t(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache,no-store")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
