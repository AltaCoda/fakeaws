package engine

import (
	"encoding/xml"
	"net/http"
)

// SES v2 JSON error shape
type jsonError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

// STS XML error shape
type stsErrorResponse struct {
	XMLName   xml.Name     `xml:"ErrorResponse"`
	Error     stsErrorBody `xml:"Error"`
	RequestID string       `xml:"RequestId"`
}

type stsErrorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

// WriteJSONError writes an SES v2 JSON error response.
func WriteJSONError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	WriteJSONResponse(w, statusCode, jsonError{
		Type:    errorCode,
		Message: message,
	})
}

// WriteXMLError writes an STS XML error response.
func WriteXMLError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	resp := stsErrorResponse{
		Error: stsErrorBody{
			Type:    "Sender",
			Code:    errorCode,
			Message: message,
		},
		RequestID: GenerateMessageID(),
	}
	WriteXMLResponse(w, statusCode, resp)
}

// WriteError writes an error response in the appropriate format for the service.
func WriteError(w http.ResponseWriter, service string, statusCode int, errorCode, message string) {
	switch service {
	case "sts":
		WriteXMLError(w, statusCode, errorCode, message)
	default:
		WriteJSONError(w, statusCode, errorCode, message)
	}
}
