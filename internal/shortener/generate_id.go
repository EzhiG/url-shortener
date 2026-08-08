package shortener

import "math/rand/v2"

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const alphabetLen = len(alphabet)
const idLength = 8

func generateId() string {
	b := make([]byte, idLength)
	for i := range idLength {
		b[i] = alphabet[rand.IntN(alphabetLen)]
	}

	return string(b)
}
