package util

import "encoding/base64"

// EncodeString encodes the given string to avoid storing as plain text
func EncodeString(inp string) string {
	inpBytes := []byte(inp)
	encodedStr := base64.StdEncoding.EncodeToString(inpBytes)
	return encodedStr
}

// DecodeString decodes the given string to plain text
func DecodeString(inp string) string {
	decodedStr, err := base64.StdEncoding.DecodeString(inp)
	if err != nil {
		return ""
	}
	return string(decodedStr)
}
