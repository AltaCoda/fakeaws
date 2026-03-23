package engine

// SES v2 route table — maps method + path pattern to operation name.
// Path parameters use {Name} syntax.
// Covers all operations used by SendOps.

var sesV2Routes = []route{
	// Sending
	{Method: "POST", Pattern: "/v2/email/outbound-emails", Operation: "SendEmail"},
	{Method: "POST", Pattern: "/v2/email/outbound-bulk-emails", Operation: "SendBulkEmail"},

	// Account
	{Method: "GET", Pattern: "/v2/email/account", Operation: "GetAccount"},
	{Method: "PUT", Pattern: "/v2/email/account/sending", Operation: "PutAccountSendingAttributes"},
	{Method: "PUT", Pattern: "/v2/email/account-details", Operation: "PutAccountDetails"},

	// Identities
	{Method: "GET", Pattern: "/v2/email/identities", Operation: "ListEmailIdentities"},
	{Method: "POST", Pattern: "/v2/email/identities", Operation: "CreateEmailIdentity"},
	{Method: "GET", Pattern: "/v2/email/identities/{EmailIdentity}", Operation: "GetEmailIdentity"},
	{Method: "DELETE", Pattern: "/v2/email/identities/{EmailIdentity}", Operation: "DeleteEmailIdentity"},
	{Method: "PUT", Pattern: "/v2/email/identities/{EmailIdentity}/configuration-set-attributes", Operation: "PutEmailIdentityConfigurationSetAttributes"},

	// Configuration sets
	{Method: "GET", Pattern: "/v2/email/configuration-sets", Operation: "ListConfigurationSets"},
	{Method: "POST", Pattern: "/v2/email/configuration-sets", Operation: "CreateConfigurationSet"},
	{Method: "GET", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}", Operation: "GetConfigurationSet"},
	{Method: "DELETE", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}", Operation: "DeleteConfigurationSet"},
	{Method: "PUT", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}/sending-options", Operation: "PutConfigurationSetSendingOptions"},
	{Method: "PUT", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}/delivery-options", Operation: "PutConfigurationSetDeliveryOptions"},
	{Method: "PUT", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}/suppression-options", Operation: "PutConfigurationSetSuppressionOptions"},
	{Method: "PUT", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}/tracking-options", Operation: "PutConfigurationSetTrackingOptions"},
	{Method: "POST", Pattern: "/v2/email/configuration-sets/{ConfigurationSetName}/event-destinations", Operation: "CreateConfigurationSetEventDestination"},

	// Templates
	{Method: "GET", Pattern: "/v2/email/templates", Operation: "ListEmailTemplates"},
	{Method: "POST", Pattern: "/v2/email/templates", Operation: "CreateEmailTemplate"},
	{Method: "GET", Pattern: "/v2/email/templates/{TemplateName}", Operation: "GetEmailTemplate"},
	{Method: "PUT", Pattern: "/v2/email/templates/{TemplateName}", Operation: "UpdateEmailTemplate"},
	{Method: "DELETE", Pattern: "/v2/email/templates/{TemplateName}", Operation: "DeleteEmailTemplate"},

	// Suppression
	{Method: "GET", Pattern: "/v2/email/suppressed-destinations", Operation: "ListSuppressedDestinations"},
	{Method: "DELETE", Pattern: "/v2/email/suppressed-destinations/{EmailAddress}", Operation: "DeleteSuppressedDestination"},
}
