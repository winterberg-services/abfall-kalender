package commands

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/klabast/wb-services/abfall-kalender/internal/app"
	"golang.org/x/term"
)

// HashPassword handles the hash-password subcommand
func HashPassword(args []string) {
	// Parse flags for hash-password subcommand
	fs := flag.NewFlagSet("hash-password", flag.ExitOnError)
	overwrite := fs.Bool("overwrite", false, "Overwrite existing auth file without asking")
	insecureUnmask := fs.Bool("insecure-unmask-password", false, "Show password as plain text (INSECURE!)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: abfall-kalender hash-password [OPTIONS]\n\n")
		fmt.Fprintf(os.Stderr, "Creates an auth.secret file with hashed password (Argon2id).\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  AUTH_FILE    Path to auth file (default: ./auth.secret)\n")
	}
	fs.Parse(args)

	// Prompt for username
	fmt.Print("Enter username: ")
	var username string
	if _, err := fmt.Scanln(&username); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading username: %v\n", err)
		os.Exit(1)
	}

	if username == "" {
		fmt.Fprintf(os.Stderr, "Username cannot be empty\n")
		os.Exit(1)
	}

	// Prompt for password
	var password, passwordConfirm string

	if *insecureUnmask {
		// Plain text mode (insecure!)
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: Password will be visible on screen!\n")
		fmt.Print("Enter password:   ")
		if _, err := fmt.Scanln(&password); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}

		fmt.Print("Confirm password: ")
		if _, err := fmt.Scanln(&passwordConfirm); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password confirmation: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Masked mode with asterisks (default, secure)
		password = readPasswordWithMask("Enter password:   ")
		passwordConfirm = readPasswordWithMask("Confirm password: ")
	}

	if password == "" {
		fmt.Fprintf(os.Stderr, "Password cannot be empty\n")
		os.Exit(1)
	}

	if password != passwordConfirm {
		fmt.Fprintf(os.Stderr, "Passwords do not match\n")
		os.Exit(1)
	}

	// Create auth file
	if err := app.CreateAuthFile(username, password, *overwrite); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// readPasswordWithMask reads password input and displays asterisks
func readPasswordWithMask(prompt string) string {
	fmt.Print(prompt)

	// Save original terminal state
	oldState, err := term.GetState(int(syscall.Stdin))
	if err != nil {
		// Fallback to hidden input if we can't set raw mode
		password, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		return string(password)
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	// Set terminal to raw mode
	if _, err := term.MakeRaw(int(syscall.Stdin)); err != nil {
		// Fallback to hidden input
		password, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		return string(password)
	}

	var password []byte
	reader := bufio.NewReader(os.Stdin)

	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		// Handle different key presses
		switch char {
		case '\n', '\r': // Enter key
			fmt.Println() // New line
			return string(password)
		case 127, 8: // Backspace or Delete
			if len(password) > 0 {
				password = password[:len(password)-1]
				// Clear the asterisk: backspace, space, backspace
				fmt.Print("\b \b")
			}
		case 3: // Ctrl+C
			fmt.Println()
			os.Exit(1)
		default:
			// Only accept printable characters
			if char >= 32 && char <= 126 {
				password = append(password, byte(char))
				fmt.Print("*")
			}
		}
	}

	fmt.Println()
	return string(password)
}
