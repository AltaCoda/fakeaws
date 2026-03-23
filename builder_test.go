package fakeaws_test

import (
	"context"
	"testing"
	"time"

	"github.com/altacoda/fakeaws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

func TestBuilder_RespondThrottle(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	fake.AddScenario(fakeaws.WhenOperation("SendEmail").RespondThrottle())

	ctx := context.Background()
	client := fake.SESClient(ctx)

	_, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("test@example.com"),
		Destination:      &types.Destination{ToAddresses: []string{"to@example.com"}},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String("Test")},
				Body:    &types.Body{Text: &types.Content{Data: aws.String("Body")}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected throttle error")
	}

	fake.AssertCalled(t, "sesv2", "SendEmail")
}

func TestBuilder_RespondMessageRejected(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	fake.AddScenario(
		fakeaws.WhenOperation("SendEmail").
			Named("bounce-test").
			RespondMessageRejected("Email address is on bounce list"),
	)

	ctx := context.Background()
	client := fake.SESClient(ctx)

	_, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("test@example.com"),
		Destination:      &types.Destination{ToAddresses: []string{"bounced@example.com"}},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String("Test")},
				Body:    &types.Body{Text: &types.Content{Data: aws.String("Body")}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected message rejected error")
	}
}

func TestBuilder_Once(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	fake.AddScenario(
		fakeaws.WhenOperation("SendEmail").Once().RespondThrottle(),
	)

	ctx := context.Background()
	client := fake.SESClient(ctx)

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("test@example.com"),
		Destination:      &types.Destination{ToAddresses: []string{"to@example.com"}},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String("Test")},
				Body:    &types.Body{Text: &types.Content{Data: aws.String("Body")}},
			},
		},
	}

	// First call: throttled
	_, err := client.SendEmail(ctx, input)
	if err == nil {
		t.Fatal("first call should be throttled")
	}

	// Second call: succeeds (Once scenario removed)
	out, err := client.SendEmail(ctx, input)
	if err != nil {
		t.Fatalf("second call should succeed: %v", err)
	}
	if out.MessageId == nil {
		t.Fatal("expected message ID on second call")
	}
}

func TestBuilder_WithField(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	// Only throttle emails to a specific address
	fake.AddScenario(
		fakeaws.WhenOperation("SendEmail").
			WithField("Destination.ToAddresses[0]", "blocked@example.com").
			RespondMessageRejected("Address is blocked"),
	)

	ctx := context.Background()
	client := fake.SESClient(ctx)

	makeInput := func(to string) *sesv2.SendEmailInput {
		return &sesv2.SendEmailInput{
			FromEmailAddress: aws.String("test@example.com"),
			Destination:      &types.Destination{ToAddresses: []string{to}},
			Content: &types.EmailContent{
				Simple: &types.Message{
					Subject: &types.Content{Data: aws.String("Test")},
					Body:    &types.Body{Text: &types.Content{Data: aws.String("Body")}},
				},
			},
		}
	}

	// blocked@example.com → rejected
	_, err := client.SendEmail(ctx, makeInput("blocked@example.com"))
	if err == nil {
		t.Fatal("expected rejection for blocked address")
	}

	// other@example.com → success
	out, err := client.SendEmail(ctx, makeInput("other@example.com"))
	if err != nil {
		t.Fatalf("expected success for other address: %v", err)
	}
	if out.MessageId == nil {
		t.Fatal("expected message ID")
	}
}

func TestBuilder_RespondAfter(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	fake.AddScenario(
		fakeaws.WhenOperation("GetAccount").
			RespondAfter(50 * time.Millisecond).
			RespondSuccess(map[string]any{"SendingEnabled": true}),
	)

	ctx := context.Background()
	client := fake.SESClient(ctx)

	start := time.Now()
	_, err := client.GetAccount(ctx, &sesv2.GetAccountInput{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected at least 50ms delay, got %v", elapsed)
	}
}

func TestBuilder_RespondTimeout(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	fake.AddScenario(
		fakeaws.WhenOperation("SendEmail").RespondTimeout(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client := fake.SESClient(ctx)

	_, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("test@example.com"),
		Destination:      &types.Destination{ToAddresses: []string{"to@example.com"}},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String("Test")},
				Body:    &types.Body{Text: &types.Content{Data: aws.String("Body")}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestBuilder_AssertFieldEquals(t *testing.T) {
	fake := fakeaws.NewFakeServer()
	defer fake.Close()

	ctx := context.Background()
	client := fake.SESClient(ctx)

	client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String("sender@example.com"),
		Destination:      &types.Destination{ToAddresses: []string{"recipient@example.com"}},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String("Hello")},
				Body:    &types.Body{Text: &types.Content{Data: aws.String("World")}},
			},
		},
	})

	reqs := fake.RequestsFor("sesv2", "SendEmail")
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}

	fake.AssertFieldEquals(t, reqs[0], "FromEmailAddress", "sender@example.com")
}
