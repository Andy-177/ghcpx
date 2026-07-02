package password

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
)

const (
	saltLen    = 16
	ivLen      = aes.BlockSize
	keyLen     = 32
	pbkdf2Iter = 100000
)

type Config struct {
	Password    string `json:"password"`
	GitHubToken string `json:"github_token"`
}

func ConfigPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "ghcp.json"), nil
}

func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keyLen, func() hash.Hash {
		return sha256.New()
	})
}

func aesEncrypt(plaintext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := make([]byte, len(plaintext)+padding)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padding)
	}

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)
	return ciphertext, nil
}

func aesDecrypt(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	padding := int(ciphertext[len(ciphertext)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	for i := len(ciphertext) - padding; i < len(ciphertext); i++ {
		if ciphertext[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return ciphertext[:len(ciphertext)-padding], nil
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	io.ReadFull(rand.Reader, b)
	return b
}

func HashPassword(password string) string {
	salt := randomBytes(saltLen)
	key := deriveKey(password, salt)
	iv := randomBytes(ivLen)
	ciphertext, err := aesEncrypt([]byte("pass"), key, iv)
	if err != nil {
		panic(err)
	}
	data := make([]byte, 0, saltLen+ivLen+len(ciphertext))
	data = append(data, salt...)
	data = append(data, iv...)
	data = append(data, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(data)
}

func VerifyPassword(hash, password string) bool {
	data, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil || len(data) < saltLen+ivLen+1 {
		return false
	}
	salt := data[:saltLen]
	iv := data[saltLen : saltLen+ivLen]
	ciphertext := data[saltLen+ivLen:]
	key := deriveKey(password, salt)

	plaintext, err := aesDecrypt(ciphertext, key, iv)
	if err != nil {
		return false
	}
	return string(plaintext) == "pass"
}

func EncryptToken(token, password, passwordHash string) (string, error) {
	data, err := base64.RawStdEncoding.DecodeString(passwordHash)
	if err != nil || len(data) < saltLen {
		return "", fmt.Errorf("invalid password hash")
	}
	salt := data[:saltLen]
	key := deriveKey(password, salt)
	iv := randomBytes(ivLen)
	ciphertext, err := aesEncrypt([]byte(token), key, iv)
	if err != nil {
		return "", err
	}
	out := make([]byte, 0, ivLen+len(ciphertext))
	out = append(out, iv...)
	out = append(out, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(out), nil
}

func DecryptToken(encrypted, password, passwordHash string) (string, error) {
	data, err := base64.RawStdEncoding.DecodeString(passwordHash)
	if err != nil || len(data) < saltLen {
		return "", fmt.Errorf("invalid password hash")
	}
	salt := data[:saltLen]
	key := deriveKey(password, salt)

	ciphertext, err := base64.RawStdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < ivLen {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:ivLen]
	ciphertext = ciphertext[ivLen:]

	plaintext, err := aesDecrypt(ciphertext, key, iv)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
