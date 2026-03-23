package engine

import (
	"encoding/xml"
	"net/http"
	"time"
)

// defaultHandlers maps operation names to their default success responders.
var defaultHandlers = map[string]Responder{
	"SendEmail":      defaultSendEmail,
	"SendBulkEmail":  defaultSendBulkEmail,
	"GetAccount":     defaultGetAccount,
	"CreateConfigurationSet": defaultEmpty,
	"GetConfigurationSet":    defaultGetConfigurationSet,
	"DeleteConfigurationSet": defaultEmpty,
	"PutAccountSendingAttributes": defaultEmpty,
	"CreateEmailIdentity":  defaultCreateEmailIdentity,
	"GetEmailIdentity":     defaultGetEmailIdentity,
	"DeleteEmailIdentity":  defaultEmpty,
	"AssumeRole":           defaultAssumeRole,
	"GetCallerIdentity":    defaultGetCallerIdentity,
}

// DefaultHandler returns the default responder for an operation, or a generic 200 OK.
func DefaultHandler(operation string) Responder {
	if h, ok := defaultHandlers[operation]; ok {
		return h
	}
	return defaultEmpty
}

func defaultEmpty(w http.ResponseWriter, req *ParsedRequest) {
	switch req.Service {
	case "sts":
		WriteXMLResponse(w, http.StatusOK, nil)
	default:
		WriteJSONResponse(w, http.StatusOK, map[string]any{})
	}
}

func defaultSendEmail(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"MessageId": GenerateMessageID(),
	})
}

func defaultSendBulkEmail(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"BulkEmailEntryResults": []map[string]any{
			{
				"Status":    "SUCCESS",
				"MessageId": GenerateMessageID(),
			},
		},
	})
}

func defaultGetAccount(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"SendQuota": map[string]any{
			"Max24HourSend":   50000.0,
			"MaxSendRate":     14.0,
			"SentLast24Hours": 100.0,
		},
		"SendingEnabled": true,
	})
}

func defaultGetConfigurationSet(w http.ResponseWriter, req *ParsedRequest) {
	name := req.PathParams["ConfigurationSetName"]
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"ConfigurationSetName": name,
	})
}

func defaultCreateEmailIdentity(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"IdentityType":             "EMAIL_ADDRESS",
		"VerifiedForSendingStatus": true,
	})
}

func defaultGetEmailIdentity(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"IdentityType":             "EMAIL_ADDRESS",
		"VerifiedForSendingStatus": true,
		"FeedbackForwardingStatus": true,
	})
}

// STS XML response types

type assumeRoleResponse struct {
	XMLName xml.Name          `xml:"AssumeRoleResponse"`
	Result  assumeRoleResult  `xml:"AssumeRoleResult"`
}

type assumeRoleResult struct {
	Credentials    stsCredentials `xml:"Credentials"`
	AssumedRoleUser assumedRoleUser `xml:"AssumedRoleUser"`
}

type stsCredentials struct {
	AccessKeyId     string `xml:"AccessKeyId"`
	SecretAccessKey  string `xml:"SecretAccessKey"`
	SessionToken    string `xml:"SessionToken"`
	Expiration      string `xml:"Expiration"`
}

type assumedRoleUser struct {
	AssumedRoleId string `xml:"AssumedRoleId"`
	Arn           string `xml:"Arn"`
}

type getCallerIdentityResponse struct {
	XMLName xml.Name                   `xml:"GetCallerIdentityResponse"`
	Result  getCallerIdentityResult    `xml:"GetCallerIdentityResult"`
}

type getCallerIdentityResult struct {
	Account string `xml:"Account"`
	Arn     string `xml:"Arn"`
	UserId  string `xml:"UserId"`
}

func defaultAssumeRole(w http.ResponseWriter, req *ParsedRequest) {
	accessKey, secretKey, sessionToken, expiration := FakeCredentials(1 * time.Hour)
	roleArn := req.FieldString("RoleArn")
	if roleArn == "" {
		roleArn = GenerateARN("iam", "role/FakeRole")
	}

	resp := assumeRoleResponse{
		Result: assumeRoleResult{
			Credentials: stsCredentials{
				AccessKeyId:    accessKey,
				SecretAccessKey: secretKey,
				SessionToken:   sessionToken,
				Expiration:     expiration.UTC().Format(time.RFC3339),
			},
			AssumedRoleUser: assumedRoleUser{
				AssumedRoleId: "AROA" + randomHex(8) + ":session",
				Arn:           roleArn,
			},
		},
	}
	WriteXMLResponse(w, http.StatusOK, resp)
}

func defaultGetCallerIdentity(w http.ResponseWriter, req *ParsedRequest) {
	resp := getCallerIdentityResponse{
		Result: getCallerIdentityResult{
			Account: FakeAccountID,
			Arn:     GenerateARN("iam", "user/FakeUser"),
			UserId:  "AIDAFAKE" + randomHex(8),
		},
	}
	WriteXMLResponse(w, http.StatusOK, resp)
}
