package handlers

import (
	"encoding/base64"

	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRBase64 generates a QR code as a base64-encoded PNG
func GenerateQRBase64(content string) (string, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(png), nil
}
