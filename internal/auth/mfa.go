package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math/big"
	"strings"

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

// GenerateBackupCodes generates a set of one-time use backup codes
// Each code is 8 characters, alphanumeric, formatted as XXXX-XXXX
func GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 10
	}

	// Characters to use (avoiding ambiguous ones like 0/O, 1/I/l)
	chars := "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	codes := make([]string, count)

	for i := 0; i < count; i++ {
		code := make([]byte, 8)
		for j := 0; j < 8; j++ {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			if err != nil {
				return nil, fmt.Errorf("failed to generate random index: %w", err)
			}
			code[j] = chars[idx.Int64()]
		}
		// Format as XXXX-XXXX
		codes[i] = string(code[:4]) + "-" + string(code[4:])
	}

	return codes, nil
}

// NormalizeBackupCode removes hyphens and converts to uppercase for comparison
func NormalizeBackupCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(code, "-", ""))
}

// ValidateBackupCode checks if a code matches any in the list and returns the remaining codes
// Returns the index of the matched code (-1 if not found) and the remaining codes
func ValidateBackupCode(inputCode string, storedCodes []string) (int, []string) {
	normalizedInput := NormalizeBackupCode(inputCode)

	for i, stored := range storedCodes {
		if NormalizeBackupCode(stored) == normalizedInput {
			// Remove the used code
			remaining := make([]string, 0, len(storedCodes)-1)
			remaining = append(remaining, storedCodes[:i]...)
			remaining = append(remaining, storedCodes[i+1:]...)
			return i, remaining
		}
	}

	return -1, storedCodes
}
