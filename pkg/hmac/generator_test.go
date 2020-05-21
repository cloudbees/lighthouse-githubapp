package hmac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHmacSignatureSha1(t *testing.T) {
	key := "this is the key"
	body := "this is a much longer message body"

	g := NewGenerator("sha1", []byte(key))

	AssertSignature(t, g, []byte(body))
}

func TestGenerateHmacSignatureSha256(t *testing.T) {
	key := "this is the key"
	body := "this is a much longer message body"

	g := NewGenerator("sha256", []byte(key))

	AssertSignature(t, g, []byte(body))
}

func AssertSignature(t *testing.T, g *Generator, body []byte) {
	signature := g.HubSignature(body)
	assert.NotEmpty(t, signature)
	t.Logf("signature = '%s'", signature)

	assert.True(t, g.VerifySignature(signature, body))
}
