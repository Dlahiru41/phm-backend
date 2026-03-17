package messaging

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	twilioClient "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type WhatsAppSender interface {
	SendOTP(ctx context.Context, toPhone, otp string, ttl time.Duration) error
}

type LogWhatsAppSender struct{}

func NewLogWhatsAppSender() *LogWhatsAppSender {
	return &LogWhatsAppSender{}
}

func (s *LogWhatsAppSender) SendOTP(_ context.Context, toPhone, otp string, ttl time.Duration) error {
	log.Printf("[whatsapp-otp] to=%s otp=%s ttl=%s", toPhone, otp, ttl)
	return nil
}

// TwilioWhatsAppSender implements WhatsAppSender using Twilio API
type TwilioWhatsAppSender struct {
	client      *twilioClient.RestClient
	phoneNumber string
}

// NewTwilioWhatsAppSender creates a new Twilio WhatsApp sender
func NewTwilioWhatsAppSender(accountSID, authToken, phoneNumber string) (*TwilioWhatsAppSender, error) {
	if accountSID == "" || authToken == "" || phoneNumber == "" {
		return nil, fmt.Errorf("twilio credentials and phone number are required")
	}

	// Sanitize phone number - remove spaces, dashes, parentheses
	sanitizedPhone := sanitizePhoneNumber(phoneNumber)
	if sanitizedPhone == "" {
		return nil, fmt.Errorf("invalid phone number format: %s", phoneNumber)
	}

	client := twilioClient.NewRestClientWithParams(twilioClient.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	log.Printf("[twilio-init] Initialized with phone number: %s (original: %s)", sanitizedPhone, phoneNumber)

	return &TwilioWhatsAppSender{
		client:      client,
		phoneNumber: sanitizedPhone,
	}, nil
}

// sanitizePhoneNumber removes spaces, dashes, and parentheses from phone numbers
// Returns phone number in format: +<country_code><number>
func sanitizePhoneNumber(phone string) string {
	// Remove all non-digit characters except leading +
	var result strings.Builder
	hasPlus := false

	for i, char := range phone {
		if char == '+' && i == 0 {
			result.WriteRune(char)
			hasPlus = true
		} else if char >= '0' && char <= '9' {
			result.WriteRune(char)
		}
	}

	sanitized := result.String()
	if sanitized == "" {
		return ""
	}

	// Ensure it starts with +
	if !hasPlus {
		sanitized = "+" + sanitized
	}

	// Validate format: should be +<digits>
	matched, _ := regexp.MatchString(`^\+\d{10,15}$`, sanitized)
	if !matched {
		log.Printf("[phone-validation] Invalid phone format: %s (sanitized: %s)", phone, sanitized)
		return ""
	}

	return sanitized
}

// SendOTP sends an OTP via WhatsApp using Twilio
func (s *TwilioWhatsAppSender) SendOTP(ctx context.Context, toPhone, otp string, ttl time.Duration) error {
	// Sanitize the recipient phone number
	sanitizedTo := sanitizePhoneNumber(toPhone)
	if sanitizedTo == "" {
		log.Printf("[whatsapp-error] invalid recipient phone format: %s", toPhone)
		return fmt.Errorf("invalid recipient phone number format: %s", toPhone)
	}

	// Format the message with TTL information
	message := fmt.Sprintf("Your verification code is: %s\nValid for %d minutes.", otp, int(ttl.Minutes()))

	// Create message parameters
	params := &openapi.CreateMessageParams{}

	// Format with "whatsapp:" prefix
	fromAddr := fmt.Sprintf("whatsapp:%s", s.phoneNumber)
	toAddr := fmt.Sprintf("whatsapp:%s", sanitizedTo)

	params.SetFrom(fromAddr)
	params.SetTo(toAddr)
	params.SetBody(message)

	log.Printf("[whatsapp-debug] From address: %s", fromAddr)
	log.Printf("[whatsapp-debug] To address: %s", toAddr)
	log.Printf("[whatsapp-debug] Phone number stored: '%s'", s.phoneNumber)
	log.Printf("[whatsapp-debug] Phone number length: %d", len(s.phoneNumber))

	// Send the message
	resp, err := s.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("[whatsapp-error] failed to send OTP to %s: %v", sanitizedTo, err)
		return fmt.Errorf("failed to send whatsapp otp: %w", err)
	}

	if resp.Sid != nil {
		log.Printf("[whatsapp-otp] successfully sent to=%s ttl=%s (SID: %s)", sanitizedTo, ttl, *resp.Sid)
	} else {
		log.Printf("[whatsapp-otp] successfully sent to=%s ttl=%s", sanitizedTo, ttl)
	}

	return nil
}
