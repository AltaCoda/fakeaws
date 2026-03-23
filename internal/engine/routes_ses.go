package engine

// SES v2 route table — maps method + path pattern to operation name.
// Path parameters use {Name} syntax.

var sesV2Routes = []route{
	{Method: "POST", Pattern: "/v2/email/outbound-emails", Operation: "SendEmail"},
	{Method: "POST", Pattern: "/v2/email/outbound-bulk-emails", Operation: "SendBulkEmail"},
	{Method: "POST", Pattern: "/v2/email/configuration-sets", Operation: "CreateConfigurationSet"},
	{Method: "GET", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}", Operation: "GetConfigurationSet"},
	{Method: "DELETE", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}", Operation: "DeleteConfigurationSet"},
	{Method: "GET", Pattern: "/v2/email/account", Operation: "GetAccount"},
	{Method: "PUT", Pattern: "/v2/email/account/sending", Operation: "PutAccountSendingAttributes"},
	{Method: "POST", Pattern: "/v2/email/identities", Operation: "CreateEmailIdentity"},
	{Method: "GET", Pattern: "/v2/email/identities/{EmailIdentity}", Operation: "GetEmailIdentity"},
	{Method: "DELETE", Pattern: "/v2/email/identities/{EmailIdentity}", Operation: "DeleteEmailIdentity"},
}
