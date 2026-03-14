package messaging

import (
	"context"
	"log"
	"time"
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
