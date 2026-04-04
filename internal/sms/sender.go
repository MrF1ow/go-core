package sms

import (
	"log"

	core "github.com/JedidiahDigital/go-core"
)

// NewSender creates the appropriate SMS sender based on the provided config.
// Returns nil if SMS is disabled (empty provider).
func NewSender(cfg core.SMSConfig) Sender {
	switch cfg.Provider {
	case ProviderTwilio:
		if cfg.TwilioAccountSID == "" || cfg.TwilioAuthToken == "" || cfg.TwilioFromNumber == "" {
			log.Println("Warning: SMS_PROVIDER=twilio but Twilio credentials are incomplete. SMS will be disabled.")
			return nil
		}
		log.Printf("SMS provider: Twilio (from: %s)", cfg.TwilioFromNumber)
		return NewTwilioSender(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber)
	case ProviderDisabled:
		log.Println("SMS provider: disabled (SMS_PROVIDER not set)")
		return nil
	default:
		log.Printf("Warning: unknown SMS_PROVIDER=%q. SMS will be disabled.", cfg.Provider)
		return nil
	}
}
