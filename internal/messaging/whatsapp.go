package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	textLKSendURL     = "https://app.text.lk/api/v3/sms/send"
	textLKBearerToken = "3955|LBMYZvFC98gojA1FhAKNNeWlPKF2DWfQmWXRd2QR4e3dd92e"
	textLKSenderID    = "TextLKDemo"
	textLKMessageType = "plain"
)

type WhatsAppSender interface {
	SendOTP(ctx context.Context, toPhone, otp string, ttl time.Duration) error
}

type LogWhatsAppSender struct{}

func NewLogWhatsAppSender() *LogWhatsAppSender {
	return &LogWhatsAppSender{}
}

func (s *LogWhatsAppSender) SendOTP(_ context.Context, toPhone, otp string, ttl time.Duration) error {
	log.Printf("[otp-log] to=%s otp=%s ttl=%s", toPhone, otp, ttl)
	return nil
}

type TextLKSender struct {
	httpClient *http.Client
}

func NewTextLKSender() *TextLKSender {
	return &TextLKSender{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type textLKSendRequest struct {
	Recipient string `json:"recipient"`
	SenderID  string `json:"sender_id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
}

// normalizeRecipient converts input to a numeric recipient acceptable by TextLK.
func normalizeRecipient(phone string) string {
	var b strings.Builder
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}

	digits := b.String()
	if len(digits) == 10 && strings.HasPrefix(digits, "0") {
		return "94" + digits[1:]
	}

	return digits
}

func (s *TextLKSender) SendOTP(ctx context.Context, toPhone, otp string, ttl time.Duration) error {
	recipient := normalizeRecipient(toPhone)
	if matched, _ := regexp.MatchString(`^\d{11,15}$`, recipient); !matched {
		return fmt.Errorf("invalid recipient phone number format: %s", toPhone)
	}

	payload := textLKSendRequest{
		Recipient: recipient,
		SenderID:  textLKSenderID,
		Type:      textLKMessageType,
		Message:   fmt.Sprintf("Your verification code is: %s. Valid for %d minutes.", otp, int(ttl.Minutes())),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to build sms payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, textLKSendURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create sms request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+textLKBearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send otp sms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("textlk api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	log.Printf("[otp-sms] successfully sent to=%s ttl=%s", recipient, ttl)
	return nil
}
