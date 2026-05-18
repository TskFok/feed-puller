package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	passwordHashVersion = "pbkdf2-sha256"
	passwordIterations  = 210000
	passwordSaltBytes   = 16
	passwordKeyBytes    = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成密码盐失败: %w", err)
	}

	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyBytes)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}

	return strings.Join([]string{
		passwordHashVersion,
		strconv.Itoa(passwordIterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, "$"), nil
}

func VerifyPassword(encodedHash, password string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 || parts[0] != passwordHashVersion {
		return false
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}

	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
