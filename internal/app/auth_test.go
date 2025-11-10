package app

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "MySecurePassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}

	// Check hash format
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("Hash should start with $argon2id$v=19$, got: %s", hash)
	}

	// Hash should be different each time (different salt)
	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() failed on second call: %v", err)
	}

	if hash == hash2 {
		t.Error("Two hashes of same password should be different (different salts)")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "MySecurePassword123"
	wrongPassword := "WrongPassword456"

	// Create hash
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "Correct password",
			password: password,
			hash:     hash,
			want:     true,
			wantErr:  false,
		},
		{
			name:     "Wrong password",
			password: wrongPassword,
			hash:     hash,
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Invalid hash format",
			password: password,
			hash:     "invalid",
			want:     false,
			wantErr:  true,
		},
		{
			name:     "Wrong algorithm",
			password: password,
			hash:     "$bcrypt$v=1$m=65536,t=1,p=4$salt$hash",
			want:     false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifyPassword(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateAuthFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	authFile := filepath.Join(tmpDir, "auth.secret")

	// Set AUTH_FILE env var
	t.Setenv("AUTH_FILE", authFile)

	username := "testuser"
	password := "TestPassword123456"

	// Test creating new file
	t.Run("Create new file", func(t *testing.T) {
		err := CreateAuthFile(username, password, false)
		if err != nil {
			t.Fatalf("CreateAuthFile() failed: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(authFile); os.IsNotExist(err) {
			t.Error("Auth file was not created")
		}

		// Verify file permissions
		info, err := os.Stat(authFile)
		if err != nil {
			t.Fatalf("Failed to stat auth file: %v", err)
		}
		if info.Mode().Perm() != 0400 {
			t.Errorf("Expected file mode 0400 (read-only), got %o", info.Mode().Perm())
		}

		// Verify content format
		content, err := os.ReadFile(authFile)
		if err != nil {
			t.Fatalf("Failed to read auth file: %v", err)
		}

		line := strings.TrimSpace(string(content))
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			t.Error("Auth file should contain username:hash")
		}

		if parts[0] != username {
			t.Errorf("Expected username %s, got %s", username, parts[0])
		}

		if !strings.HasPrefix(parts[1], "$argon2id$") {
			t.Error("Hash should be Argon2id format")
		}

		// Verify password can be verified
		match, err := VerifyPassword(password, parts[1])
		if err != nil {
			t.Fatalf("VerifyPassword() failed: %v", err)
		}
		if !match {
			t.Error("Password verification failed for created hash")
		}
	})

	// Test overwrite with flag
	t.Run("Overwrite with flag", func(t *testing.T) {
		err := CreateAuthFile("newuser", "NewPassword123456", true)
		if err != nil {
			t.Fatalf("CreateAuthFile() with overwrite failed: %v", err)
		}

		content, _ := os.ReadFile(authFile)
		if !strings.HasPrefix(string(content), "newuser:") {
			t.Error("File should be overwritten with new username")
		}
	})
}

func TestLoadAuthCredentials(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(string) error
		wantUser    string
		wantErr     bool
		wantAuthNil bool
	}{
		{
			name: "Valid auth file",
			setupFile: func(path string) error {
				hash, _ := HashPassword("TestPassword123456")
				return os.WriteFile(path, []byte("testuser:"+hash), 0600)
			},
			wantUser:    "testuser",
			wantErr:     false,
			wantAuthNil: false,
		},
		{
			name: "File not exists (dev mode)",
			setupFile: func(path string) error {
				return nil // Don't create file
			},
			wantUser:    "",
			wantErr:     false,
			wantAuthNil: true,
		},
		{
			name: "Invalid format (missing colon)",
			setupFile: func(path string) error {
				return os.WriteFile(path, []byte("invalidformat"), 0600)
			},
			wantUser:    "",
			wantErr:     true,
			wantAuthNil: true,
		},
		{
			name: "Invalid format (empty)",
			setupFile: func(path string) error {
				return os.WriteFile(path, []byte(""), 0600)
			},
			wantUser:    "",
			wantErr:     true,
			wantAuthNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			authFile := filepath.Join(tmpDir, "auth.secret")

			// Set AUTH_FILE env var
			t.Setenv("AUTH_FILE", authFile)

			// Setup file
			if err := tt.setupFile(authFile); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Reset global vars
			EditUser = ""
			authHash = nil

			// Load credentials
			err := LoadAuthCredentials()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAuthCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if EditUser != tt.wantUser {
				t.Errorf("EditUser = %s, want %s", EditUser, tt.wantUser)
			}

			if (authHash == nil) != tt.wantAuthNil {
				t.Errorf("authHash nil = %v, want %v", authHash == nil, tt.wantAuthNil)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	// Setup test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	// Create a valid hash for testing
	password := "TestPassword123456"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to create test hash: %v", err)
	}

	tests := []struct {
		name           string
		setupAuth      func()
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid credentials",
			setupAuth: func() {
				EditUser = "admin"
				authHash = []byte(hash)
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:"+password)),
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name: "Invalid password",
			setupAuth: func() {
				EditUser = "admin"
				authHash = []byte(hash)
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword")),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
		{
			name: "Invalid username",
			setupAuth: func() {
				EditUser = "admin"
				authHash = []byte(hash)
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("wronguser:"+password)),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
		{
			name: "No auth header",
			setupAuth: func() {
				EditUser = "admin"
				authHash = []byte(hash)
			},
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
		{
			name: "Dev mode (no auth file)",
			setupAuth: func() {
				EditUser = ""
				authHash = nil
			},
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupAuth()

			req := httptest.NewRequest("GET", "/edit", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler := RequireAuth(testHandler)
			handler(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body := w.Body.String()
			if body != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, body)
			}

			// Check WWW-Authenticate header on 401
			if tt.expectedStatus == http.StatusUnauthorized {
				authHeader := resp.Header.Get("WWW-Authenticate")
				if authHeader == "" {
					t.Error("Expected WWW-Authenticate header on 401")
				}
			}
		})
	}
}

func TestArgon2idParameters(t *testing.T) {
	// Test that our Argon2id parameters are reasonable
	if argon2Memory < 64*1024 {
		t.Error("Argon2id memory should be at least 64MB (OWASP recommendation)")
	}

	if argon2Time < 1 {
		t.Error("Argon2id time parameter should be at least 1")
	}

	if argon2Threads < 1 {
		t.Error("Argon2id threads should be at least 1")
	}

	if argon2KeyLen < 32 {
		t.Error("Argon2id key length should be at least 32 bytes")
	}

	if saltLen < 16 {
		t.Error("Salt length should be at least 16 bytes")
	}
}
