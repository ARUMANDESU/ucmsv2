package randcode

import (
	"crypto/rand"
	"math/big"
)

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func GenerateAlphaNumericCode(length int) (string, error) {
	b := make([]rune, length)

	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}

		b[i] = letters[n.Int64()]
	}

	return string(b), nil
}
