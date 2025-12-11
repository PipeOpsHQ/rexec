package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"

	"github.com/pquerna/otp/totp"
)

// MFAService handles TOTP-based MFA
type MFAService struct {
	issuer string
}

// NewMFAService creates a new MFA service
func NewMFAService(issuer string) *MFAService {
	return &MFAService{
		issuer: issuer,
	}
}

// GenerateSecret generates a new TOTP secret and QR code URL
func (s *MFAService) GenerateSecret(accountName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", err
	}

	return key.Secret(), nil
}

// GetOTPURL returns the otpauth URL for the secret
func (s *MFAService) GetOTPURL(accountName, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", s.issuer, accountName, secret, s.issuer)
}

// Validate validates a TOTP code
func (s *MFAService) Validate(passcode, secret string) bool {
	return totp.Validate(passcode, secret)
}

// GenerateRandomSecret generates a random base32 secret
func GenerateRandomSecret() (string, error) {
	randomBytes := make([]byte, 20)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes), nil
}