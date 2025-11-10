package app

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Auth configuration
var (
	EditUser       string
	authSecretFile string
	authHash       []byte
)

const (
	DefaultAuthFile = "auth.secret"
	ErrNoAuthFile   = "No auth.secret file found"
)

// Argon2id parameters (OWASP recommended)
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLen       = 16
)

// LoadAuthCredentials loads auth credentials from file
func LoadAuthCredentials() error {
	// Determine auth file path
	authSecretFile = os.Getenv("AUTH_FILE")
	if authSecretFile == "" {
		// Default: auth.secret in same directory as binary
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		authSecretFile = filepath.Join(filepath.Dir(execPath), DefaultAuthFile)
	}

	// Try to read auth file
	data, err := os.ReadFile(authSecretFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("╔══════════════════════════════════════════════════════════════════╗")
			log.Println("║                         ⚠️  WARNING ⚠️                            ║")
			log.Println("║                                                                  ║")
			log.Println("║  NO AUTH FILE FOUND - EDIT MODE UNPROTECTED!                    ║")
			log.Println("║                                                                  ║")
			log.Println("║  This is for LOCAL DEVELOPMENT ONLY!                            ║")
			log.Println("║  DO NOT USE IN PRODUCTION!                                      ║")
			log.Println("║                                                                  ║")
			log.Printf("║  Expected file: %-47s ║\n", authSecretFile)
			log.Println("║                                                                  ║")
			log.Println("║  To create auth file, run:                                      ║")
			log.Println("║    ./abfall-kalender hash-password                              ║")
			log.Println("║                                                                  ║")
			log.Println("╚══════════════════════════════════════════════════════════════════╝")
			return nil
		}
		return fmt.Errorf("failed to read auth file: %w", err)
	}

	// Parse auth file (format: username:hash)
	line := strings.TrimSpace(string(data))
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid auth file format (expected: username:hash)")
	}

	EditUser = parts[0]
	authHash = []byte(parts[1])

	log.Printf("✅ Basic Auth enabled for edit mode (user: %s, file: %s)", EditUser, authSecretFile)
	return nil
}

// HashPassword creates an Argon2id hash of the password
func HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password with Argon2id
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode as: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads, b64Salt, b64Hash), nil
}

// VerifyPassword verifies a password against an Argon2id hash
func VerifyPassword(password, hash string) (bool, error) {
	// Parse hash format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, fmt.Errorf("not an argon2id hash")
	}

	// Parse parameters
	var memory, time, threads uint32
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("failed to parse hash parameters: %w", err)
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the provided password with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, uint8(threads), uint32(len(decodedHash)))

	// Compare using constant-time comparison
	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1, nil
}

// RequireAuth is a middleware that enforces Basic Auth with Argon2id
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no auth file loaded, skip auth (dev mode)
		if authHash == nil {
			next(w, r)
			return
		}

		// Get credentials from request
		user, pass, ok := r.BasicAuth()

		// Check username with constant-time comparison
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(EditUser)) == 1

		// Verify password with Argon2id
		passMatch := false
		if ok && userMatch {
			var err error
			passMatch, err = VerifyPassword(pass, string(authHash))
			if err != nil {
				log.Printf("Error verifying password: %v", err)
				passMatch = false
			}
		}

		if !ok || !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="Abfallkalender Edit Mode"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("⚠️  Failed auth attempt from %s (user: %s)", r.RemoteAddr, user)
			return
		}

		next(w, r)
	}
}

// CreateAuthFile creates an auth.secret file with username and hashed password
func CreateAuthFile(username, password string, overwrite bool) error {
	// Determine auth file path
	authFile := os.Getenv("AUTH_FILE")
	if authFile == "" {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		authFile = filepath.Join(filepath.Dir(execPath), DefaultAuthFile)
	}

	// Check if file exists
	if _, err := os.Stat(authFile); err == nil {
		if !overwrite {
			fmt.Printf("Auth file already exists: %s\n", authFile)
			fmt.Print("Overwrite? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("aborted")
			}
		}
		// Delete existing file (necessary because we use 0400 read-only)
		if err := os.Remove(authFile); err != nil {
			return fmt.Errorf("failed to remove existing auth file: %w", err)
		}
	}

	// Hash password
	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Write to file with format: username:hash (0400 = read-only)
	content := fmt.Sprintf("%s:%s\n", username, hash)
	if err := os.WriteFile(authFile, []byte(content), 0400); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	fmt.Printf("✅ Auth file created: %s (mode: 0400 read-only)\n", authFile)
	fmt.Printf("   Username: %s\n", username)
	return nil
}
