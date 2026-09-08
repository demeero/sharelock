package share

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/demeero/sharelock/internal/errbrick"
)

func TestEncode_Decode_RoundTrip(t *testing.T) {
	value := []byte("0123456789abcdef")

	decoded, err := Decode(Encode(value), uint(len(value)))

	require.NoError(t, err)
	assert.Equal(t, value, decoded)
}

func TestDecode_RejectsInvalidBase64(t *testing.T) {
	_, err := Decode("not base64!!", 16)

	require.ErrorIs(t, err, errbrick.ErrInvalidData)
}

func TestDecode_RejectsSizeMismatch(t *testing.T) {
	value := []byte("0123456789abcdef")

	_, err := Decode(Encode(value), uint(len(value))+1)

	require.ErrorIs(t, err, errbrick.ErrInvalidData)
}
