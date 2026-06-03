package utils

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

func Base64Encode(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func Base64Decode(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func TimeFromID(id string) string {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) == 0 {
		return "-"
	}

	ms, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "-"
	}

	return time.UnixMilli(ms).Format("2006-01-02 15:04:05.000 -07:00 MST")
}

func ExtractMapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func ParseExpiredKey(s string) (stream string, group string, name string) {
	parts := strings.SplitN(s, ":", 5)
	return parts[2], parts[3], parts[4]
}
