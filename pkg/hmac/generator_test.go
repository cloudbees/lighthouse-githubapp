package hmac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHmacSignature(t *testing.T) {
	key := "this is the key"
	body := "this is a much longer message body"

	AssertSignature(t, []byte(key), []byte(body))
}

func AssertSignature(t *testing.T, secret, body []byte) {
	g := NewGenerator(secret)

	signature := g.HubSignature(body)
	assert.NotEmpty(t, signature)
	t.Logf("signature = '%s'", signature)

	assert.True(t, g.VerifySignature(signature, body))
}
