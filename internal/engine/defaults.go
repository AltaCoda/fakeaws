package engine

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// defaultHandlers maps operation names to their default success responders.
var defaultHandlers = map[string]Responder{
	// Sending
	"SendEmail":     defaultSendEmail,
	"SendBulkEmail": defaultSendBulkEmail,

	// Account
	"GetAccount":                  defaultGetAccount,
	"PutAccountSendingAttributes": defaultEmpty,
	"PutAccountDetails":           defaultEmpty,

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
		WriteJSONResponse(w, http.StatusOK, struct{}{})
	}
}

// --- Sending ---

func defaultSendEmail(w http.ResponseWriter, req *ParsedRequest) {
	msgID := GenerateMessageID()
	WriteJSONResponse(w, http.StatusOK, sesv2.SendEmailOutput{
		MessageId: &msgID,
	})
}

func defaultSendBulkEmail(w http.ResponseWriter, req *ParsedRequest) {
	msgID := GenerateMessageID()
	WriteJSONResponse(w, http.StatusOK, sesv2.SendBulkEmailOutput{
		BulkEmailEntryResults: []sestypes.BulkEmailEntryResult{
			{
				Status:    sestypes.BulkEmailStatusSuccess,
				MessageId: &msgID,
			},
		},
	})
}

// --- Account ---

func defaultGetAccount(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, sesv2.GetAccountOutput{
		SendingEnabled:          true,
		ProductionAccessEnabled: true,
		EnforcementStatus:       aws.String("HEALTHY"),
		SendQuota: &sestypes.SendQuota{
			Max24HourSend:   50000,
			MaxSendRate:     14,
			SentLast24Hours: 100,
		},
		Details: &sestypes.AccountDetails{
			WebsiteURL: aws.String("https://sendops.dev"),
		},
	})
}

// --- Identities ---

func defaultListEmailIdentities(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, sesv2.ListEmailIdentitiesOutput{
		EmailIdentities: []sestypes.IdentityInfo{},
	})
}

func defaultCreateEmailIdentity(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, sesv2.CreateEmailIdentityOutput{
		IdentityType:             sestypes.IdentityTypeEmailAddress,
		VerifiedForSendingStatus: true,
		DkimAttributes: &sestypes.DkimAttributes{
			SigningEnabled: true,
			Status:         sestypes.DkimStatusSuccess,
			Tokens:         []string{"token1", "token2", "token3"},
		},
	})
}

func defaultGetEmailIdentity(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, sesv2.GetEmailIdentityOutput{
		IdentityType:             sestypes.IdentityTypeEmailAddress,
		VerifiedForSendingStatus: true,
		FeedbackForwardingStatus: true,
		DkimAttributes: &sestypes.DkimAttributes{
			SigningEnabled: true,
			Status:         sestypes.DkimStatusSuccess,
			Tokens:         []string{"token1", "token2", "token3"},
		},
	})
}

// --- Configuration Sets ---

func defaultListConfigurationSets(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, sesv2.ListConfigurationSetsOutput{
		ConfigurationSets: []string{},
	})
}

func defaultGetConfigurationSet(w http.ResponseWriter, req *ParsedRequest) {
	name := req.PathParams["ConfigurationSetName"]
	WriteJSONResponse(w, http.StatusOK, sesv2.GetConfigurationSetOutput{
		ConfigurationSetName: &name,
	})
}

// --- Templates ---

func defaultListEmailTemplates(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, sesv2.ListEmailTemplatesOutput{
		TemplatesMetadata: []sestypes.EmailTemplateMetadata{},
	})
}

func defaultGetEmailTemplate(w http.ResponseWriter, req *ParsedRequest) {
	name := req.PathParams["TemplateName"]
	WriteJSONResponse(w, http.StatusOK, sesv2.GetEmailTemplateOutput{
		TemplateName: &name,
		TemplateContent: &sestypes.EmailTemplateContent{
			Subject: aws.String("Template " + name),
			Html:    aws.String("<p>Template content</p>"),
		},
	})
}

// --- Suppression ---

func defaultListSuppressedDestinations(w http.ResponseWriter, req *ParsedRequest) {
	WriteJSONResponse(w, http.StatusOK, sesv2.ListSuppressedDestinationsOutput{
		SuppressedDestinationSummaries: []sestypes.SuppressedDestinationSummary{},
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
