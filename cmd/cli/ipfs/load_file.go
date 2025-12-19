package ipfs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newLoadFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load-file <path>",
		Short: "Add a local file to IPFS and pin it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			cmd.Printf("Loading %s into IPFS …\n", filePath)
			
			// Get file size for display
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				cmd.PrintErrf("Failed to get file info: %v\n", err)
				return err
			}
			cmd.Printf("File size: %d bytes\n", fileInfo.Size())
			
			// TODO: add symmetric key management
			// // Generate a random AES256 key
			// key := make([]byte, 32) // AES256 requires 32 bytes
			// if _, err := rand.Read(key); err != nil {
			// 	cmd.PrintErrf("Failed to generate encryption key: %v\n", err)
			// 	return err
			// }
			
			// // Encrypt the file content
			// encryptedContent, err := encryptAES256(fileContent, key)
			// if err != nil {
			// 	cmd.PrintErrf("Failed to encrypt file: %v\n", err)
			// 	return err
			// }
			
			// cmd.Printf("Encrypted content size: %d bytes\n", len(encryptedContent))
			
			// Load the file into IPFS using the utility function
			ipfsPath, err := LoadFileToIPFS(filePath)
			if err != nil {
				cmd.PrintErrf("Failed to load file to IPFS: %v\n", err)
				return err
			}
			
			cmd.Printf("✅ File successfully uploaded to IPFS\n")
			cmd.Printf("📁 IPFS Path: %s\n", ipfsPath)
			// cmd.Printf("🔑 Encryption Key (hex): %x\n", key)
			// cmd.Printf("⚠️  Store this key securely to decrypt the file later!\n")
			
			return nil
		},
	}
}

// encryptAES256 encrypts data using AES256-GCM
func encryptAES256(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}
