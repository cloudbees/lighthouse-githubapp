package hmac

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

type Generator struct {
	secret []byte
}

func NewGenerator(s []byte) *Generator {
	return &Generator{ s}
}

func (g *Generator) SignBody(body []byte) []byte {
	computed := hmac.New(sha1.New, g.secret)
	computed.Write(body)
	return []byte(computed.Sum(nil))
}

func (g *Generator) HubSignature(body []byte) string {
	signature := g.SignBody(body)
	signatureString := fmt.Sprintf("%x", signature)
	return fmt.Sprintf("sha1=%s", signatureString)
}

func (g *Generator) VerifySignature(signature string, body []byte) bool {
	const signaturePrefix = "sha1="
	const signatureLength = 45 // len(SignaturePrefix) + len(hex(sha1))

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}

	actual := make([]byte, 20)
	hex.Decode(actual, []byte(signature[5:]))

	return hmac.Equal(g.SignBody(body), actual)
}
