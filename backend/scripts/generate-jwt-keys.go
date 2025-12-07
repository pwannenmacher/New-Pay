package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	// Generate ECDSA P-256 key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate key: %v\n", err)
		os.Exit(1)
	}

	// Encode private key to PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal private key: %v\n", err)
		os.Exit(1)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	fmt.Println("Generated ECDSA P-256 key pair for JWT signing.")
	fmt.Println("\nAdd this to your .env file as JWT_SECRET (as a single line with \\n for newlines):")
	fmt.Println("----------------------------------------")

	// Print as single line with escaped newlines for .env
	singleLine := ""
	for _, line := range string(privateKeyPEM) {
		if line == '\n' {
			singleLine += "\\n"
		} else {
			singleLine += string(line)
		}
	}
	fmt.Printf("JWT_SECRET=%s\n", singleLine)

	fmt.Println("\nOr save to files:")
	fmt.Println("----------------------------------------")

	// Save to file
	if err := os.WriteFile("jwt-private-key.pem", privateKeyPEM, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write private key file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Private key saved to: jwt-private-key.pem")
	fmt.Println("\nTo use the file-based key, set in .env:")
	fmt.Println("JWT_SECRET=$(cat jwt-private-key.pem)")
}
