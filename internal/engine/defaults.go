package engine

import (
	"encoding/xml"
	"net/http"
	"time"
)

// defaultHandlers maps operation names to their default success responders.
var defaultHandlers = map[string]Responder{
	// Sending
	"SendEmail":     defaultSendEmail,
	"SendBulkEmail": defaultSendBulkEmail,

	// Account
	"GetAccount":               defaultGetAccount,
	"PutAccountSendingAttributes": defaultEmpty,
	"PutAccountDetails":        defaultEmpty,

	// Identities
	"ListEmailIdentities":                      defaultListEmailIdentities,
	"CreateEmailIdentity":                       defaultCreateEmailIdentity,
	"GetEmailIdentity":                          defaultGetEmailIdentity,
	"DeleteEmailIdentity":                       defaultEmpty,
	"PutEmailIdentityConfigurationSetAttributes": defaultEmpty,

	// Configuration sets
	"ListConfigurationSets":                 defaultListConfigurationSets,
	"CreateConfigurationSet":                defaultEmpty,
	"GetConfigurationSet":                   defaultGetConfigurationSet,
	"DeleteConfigurationSet":                defaultEmpty,
	"PutConfigurationSetSendingOptions":     defaultEmpty,
	"PutConfigurationSetDeliveryOptions":    defaultEmpty,
	"PutConfigurationSetSuppressionOptions": defaultEmpty,
	"PutConfigurationSetTrackingOptions":    defaultEmpty,
	"CreateConfigurationSetEventDestination": defaultEmpty,

	// Templates
	"ListEmailTemplates":  defaultListEmailTemplates,
	"CreateEmailTemplate": defaultEmpty,
	"GetEmailTemplate":    defaultGetEmailTemplate,
	"UpdateEmailTemplate": defaultEmpty,
	"DeleteEmailTemplate": defaultEmpty,

	// Suppression
	"ListSuppressedDestinations":  defaultListSuppressedDestinations,
	"DeleteSuppressedDestination": defaultEmpty,

	// STS
	"AssumeRole":        defaultAssumeRole,
	"GetCallerIdentity": defaultGetCallerIdentity,
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

// --- Sending ---

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

// --- Account ---

func defaultGetAccount(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"SendQuota": map[string]any{
			"Max24HourSend":   50000.0,
			"MaxSendRate":     14.0,
			"SentLast24Hours": 100.0,
		},
		"SendingEnabled":          true,
		"ProductionAccessEnabled": true,
		"EnforcementStatus":       "HEALTHY",
		"Details": map[string]any{
			"WebsiteURL": "https://sendops.dev",
		},
	})
}

// --- Identities ---

func defaultListEmailIdentities(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"EmailIdentities": []map[string]any{},
	})
}

func defaultCreateEmailIdentity(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"IdentityType":             "EMAIL_ADDRESS",
		"VerifiedForSendingStatus": true,
		"DkimAttributes": map[string]any{
			"SigningEnabled": true,
			"Status":         "SUCCESS",
			"Tokens":         []string{"token1", "token2", "token3"},
		},
	})
}

func defaultGetEmailIdentity(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"IdentityType":             "EMAIL_ADDRESS",
		"VerifiedForSendingStatus": true,
		"FeedbackForwardingStatus": true,
		"DkimAttributes": map[string]any{
			"SigningEnabled": true,
			"Status":         "SUCCESS",
			"Tokens":         []string{"token1", "token2", "token3"},
		},
	})
}

// --- Configuration Sets ---

func defaultListConfigurationSets(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"ConfigurationSets": []map[string]any{},
	})
}

func defaultGetConfigurationSet(w http.ResponseWriter, req *ParsedRequest) {
	name := req.PathParams["ConfigurationSetName"]
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"ConfigurationSetName": name,
	})
}

// --- Templates ---

func defaultListEmailTemplates(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"TemplatesMetadata": []map[string]any{},
	})
}

func defaultGetEmailTemplate(w http.ResponseWriter, req *ParsedRequest) {
	name := req.PathParams["TemplateName"]
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"TemplateName": name,
		"TemplateContent": map[string]any{
			"Subject": "Template " + name,
			"Html":    "<p>Template content</p>",
		},
	})
}

// --- Suppression ---

func defaultListSuppressedDestinations(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"SuppressedDestinationSummaries": []map[string]any{},
	})
}

// --- STS ---

type assumeRoleResponse struct {
	XMLName xml.Name         `xml:"AssumeRoleResponse"`
	Result  assumeRoleResult `xml:"AssumeRoleResult"`
}

type assumeRoleResult struct {
	Credentials     stsCredentials  `xml:"Credentials"`
	AssumedRoleUser assumedRoleUser `xml:"AssumedRoleUser"`
}

type stsCredentials struct {
	AccessKeyId    string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	SessionToken   string `xml:"SessionToken"`
	Expiration     string `xml:"Expiration"`
}

type assumedRoleUser struct {
	AssumedRoleId string `xml:"AssumedRoleId"`
	Arn           string `xml:"Arn"`
}

type getCallerIdentityResponse struct {
	XMLName xml.Name                `xml:"GetCallerIdentityResponse"`
	Result  getCallerIdentityResult `xml:"GetCallerIdentityResult"`
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
