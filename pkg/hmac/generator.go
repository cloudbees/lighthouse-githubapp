package hmac

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/jenkins-x/jx-logging/pkg/log"
)

type Generator struct {
	algo string
	secret []byte
}

func NewGenerator(algo string, s []byte) *Generator {
	return &Generator{algo: algo, secret: s}
}

func (g *Generator) SignBody(body []byte) []byte {
	var computed hash.Hash
	switch g.algo {
	case "sha1":
		computed = hmac.New(sha1.New, g.secret)
	case "sha256":
		computed = hmac.New(sha256.New, g.secret)
	default:
		panic("unknown algorithum")
	}

	_, err := computed.Write(body)
	if err != nil {
		log.Logger().Errorf("unable to write to hmac: %s", err)
	}
	return computed.Sum(nil)
}

func (g *Generator) HubSignature(body []byte) string {
	signature := g.SignBody(body)
	signatureString := fmt.Sprintf("%x", signature)
	return fmt.Sprintf("%s=%s", g.algo, signatureString)
}

func (g *Generator) VerifySignature(signature string, body []byte) bool {
	signaturePrefix := fmt.Sprintf("%s=", g.algo)

	var signatureLength int // len(SignaturePrefix) + len(hex(sha1))
	var actual []byte
	switch g.algo {
	case "sha1":
		signatureLength = 45
		actual = make([]byte, 20)
	case "sha256":
		signatureLength = 71
		actual = make([]byte, 32)
	default:
		panic("unknown algorithum")
	}

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}

	_, err := hex.Decode(actual, []byte(signature[len(signaturePrefix):]))
	if err != nil {
		log.Logger().Errorf("unable to decode: %s", err)
		return false
	}
	return hmac.Equal(g.SignBody(body), actual)
}
