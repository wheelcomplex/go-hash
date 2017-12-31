package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"io"
	"math/big"
	"runtime"

	// FIXME use golang argon2 when this bug fix is released: https://github.com/golang/go/issues/23245
	//"github.com/golang/crypto/argon2"
	"github.com/aead/argon2"
)

const (
	// TIME complexity parameter for Argon2, used by PasswordHash.
	TIME uint32 = 8

	// MEMORY complexity parameter for Argon2, used by PasswordHash.
	MEMORY uint32 = 32 * 1024

	// SALTLEN length of salt given by GenerateSalt.
	SALTLEN uint32 = 32

	// KEYLEN length of key generated by PasswordHash.
	KEYLEN uint32 = 32 // 32-bytes keys are used with AES-256
)

// THREADS number of Threads to use in PasswordHash.
var THREADS = uint8(runtime.NumCPU())

var defaultPasswordCharRange []uint8

func init() {
	var (
		minChar uint8 = ' '
		maxChar uint8 = '~'
	)
	charRange := make([]uint8, 1+maxChar-minChar)
	for i := 0; i < len(charRange); i++ {
		charRange[i] = minChar + uint8(i)
	}

	defaultPasswordCharRange = charRange
}

// Encrypt a message given a secret key.
func Encrypt(key, message []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(message))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], message)
	return ciphertext, nil
}

// Decrypt a message given a secret key.
func Decrypt(key, message []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(message) < aes.BlockSize {
		return nil, errors.New("Invalid ciphertext")
	}
	iv := message[:aes.BlockSize]
	message = message[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(message, message)
	return message, nil
}

// Hmac of the message based on the given key.
func Hmac(key, message []byte) []byte {
	mac := hmac.New(sha512.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// VerifyHmac verifies the two given HMAC's are equal.
// Returns true if equal, false otherwise.
func VerifyHmac(mac1, mac2 []byte) bool {
	return hmac.Equal(mac1, mac2)
}

// GenerateSalt generates a random sequence of bytes that can be used as
// a password salt.
func GenerateSalt() []byte {
	return GenerateRandomBytes(SALTLEN)
}

// GenerateRandomBytes generates a random sequence of bytes.
func GenerateRandomBytes(len uint32) []byte {
	result := make([]byte, len)
	if _, err := io.ReadFull(rand.Reader, result); err != nil {
		panic(err)
	}
	return result
}

// DefaultPasswordCharRange returns the default ASCII characters to be used with GeneratePassword().
func DefaultPasswordCharRange() []uint8 {
	return defaultPasswordCharRange
}

// GeneratePassword generates a random sequence of the given ASCII characters.
func GeneratePassword(length int, characters []uint8) string {
	maxIndex := len(characters)
	if maxIndex < 2 {
		panic("At least 2 characters must be provided")
	}
	maxIndexBig := big.NewInt(int64(maxIndex))
	result := make([]uint8, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, maxIndexBig)
		if err != nil {
			panic(err)
		}
		result[i] = characters[int(n.Uint64())]
	}
	return string(result)
}

// PasswordHash creates a cryptographical hash of the salted password.
func PasswordHash(password string, salt []byte) []byte {
	return argon2.Key([]byte(password), salt, TIME, MEMORY, THREADS, KEYLEN)
}

// CheckSum checksum of the message.
func CheckSum(message []byte) []byte {
	hash := sha512.New()
	if _, err := hash.Write(message); err != nil {
		panic(err)
	}
	return hash.Sum(nil)
}
