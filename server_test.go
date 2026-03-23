package fakeaws_test

import (
	"context"
	"testing"

	"github.com/altacoda/fakeaws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func TestFakeServer_SES_SendEmail_RoundTrip(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	ctx := context.Background()
	client := fake.SESClient(ctx)

	out, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination: &types.Destination{
			ToAddresses: []string{"recipient@example.com"},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String("Test Subject")},
				Body:    &types.Body{Text: &types.Content{Data: aws.String("Hello!")}},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}
	if out.MessageId == nil || *out.MessageId == "" {
		t.Fatal("expected a message ID")
	}

	fake.AssertCalled(t, "sesv2", "SendEmail")
	fake.AssertCalledN(t, "sesv2", "SendEmail", 1)
	fake.AssertNotCalled(t, "sesv2", "GetAccount")
}

func TestFakeServer_SES_GetAccount_RoundTrip(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	ctx := context.Background()
	client := fake.SESClient(ctx)

	out, err := client.GetAccount(ctx, &sesv2.GetAccountInput{})
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if !out.SendingEnabled {
		t.Fatal("expected SendingEnabled=true")
	}

	fake.AssertCalled(t, "sesv2", "GetAccount")
}

func TestFakeServer_STS_AssumeRole_RoundTrip(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	ctx := context.Background()
	client := fake.STSClient(ctx)

	out, err := client.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String("arn:aws:iam::123456789012:role/TestRole"),
		RoleSessionName: aws.String("test-session"),
	})
	if err != nil {
		t.Fatalf("AssumeRole failed: %v", err)
	}
	if out.Credentials == nil {
		t.Fatal("expected credentials")
	}
	if out.Credentials.AccessKeyId == nil || *out.Credentials.AccessKeyId == "" {
		t.Fatal("expected access key ID")
	}

	fake.AssertCalled(t, "sts", "AssumeRole")
}

func TestFakeServer_STS_GetCallerIdentity_RoundTrip(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	ctx := context.Background()
	client := fake.STSClient(ctx)

	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		t.Fatalf("GetCallerIdentity failed: %v", err)
	}
	if out.Account == nil || *out.Account != fakeaws.FakeAccountID {
		t.Fatalf("expected account %s, got %v", fakeaws.FakeAccountID, out.Account)
	}

	fake.AssertCalled(t, "sts", "GetCallerIdentity")
}

func TestFakeServer_MultipleConcurrent(t *testing.T) {
	fake1 := fakeaws.NewFakeServer()
	defer fake1.Close()
	fake2 := fakeaws.NewFakeServer()
	defer fake2.Close()

	if fake1.URL() == fake2.URL() {
		t.Fatal("two servers should use different ports")
	}
}

func TestFakeServer_Reset(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	ctx := context.Background()
	client := fake.SESClient(ctx)

	client.GetAccount(ctx, &sesv2.GetAccountInput{})
	fake.AssertCalled(t, "sesv2", "GetAccount")

	fake.Reset()
	fake.AssertNotCalled(t, "sesv2", "GetAccount")
}
