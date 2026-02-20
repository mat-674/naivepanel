package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePassword checks a password against a bcrypt hash
func ComparePassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken creates a JWT token with the given admin ID
func GenerateToken(adminID int64, secret string) (string, error) {
	claims := jwt.MapClaims{
		"admin_id": adminID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates a JWT token and returns the admin ID
func ValidateToken(tokenString, secret string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	adminID, ok := claims["admin_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid admin_id in token")
	}

	return int64(adminID), nil
}
