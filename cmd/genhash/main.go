package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/genhash <password>")
		os.Exit(1)
	}

	plainPass := os.Args[1]
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPass), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("REMINDERIN_PASSWORD_HASH=%s\n", string(hash))
}
