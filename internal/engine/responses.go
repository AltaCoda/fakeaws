package engine

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
)

// WriteJSONResponse writes a JSON response with the given status code.
func WriteJSONResponse(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-amzn-RequestId", GenerateMessageID())
	w.WriteHeader(statusCode)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// WriteXMLResponse writes an XML response with the given status code.
func WriteXMLResponse(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("x-amzn-RequestId", GenerateMessageID())
	w.WriteHeader(statusCode)
	if body != nil {
		w.Write([]byte(xml.Header))
		_ = xml.NewEncoder(w).Encode(body)
	}
}
