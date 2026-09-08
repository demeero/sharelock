package share

import (
	"encoding/base64"
	"fmt"

	"github.com/demeero/sharelock/internal/errbrick"
)

func Encode(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func Decode(value string, identifierSize uint) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%w: corrupted", errbrick.ErrInvalidData)
	}
	if len(decoded) != int(identifierSize) {
		return nil, fmt.Errorf("%w: size mismatch", errbrick.ErrInvalidData)
	}

	return decoded, nil
}
