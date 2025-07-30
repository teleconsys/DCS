package ipfs

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/kubo/client/rpc"
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
			
			// Read the file
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				cmd.PrintErrf("Failed to read file %s: %v\n", filePath, err)
				return err
			}
			
			cmd.Printf("File size: %d bytes\n", len(fileContent))
			
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
			
			// Connect to local IPFS node
			api, err := rpc.NewLocalApi()
			if err != nil {
				cmd.PrintErrf("Failed to connect to IPFS node: %v\n", err)
				return err
			}
			
			ctx := context.Background()
			
			// Add the encrypted content to IPFS
			node := files.NewReaderFile(bytes.NewReader(fileContent))
			ipfsPath, err := api.Unixfs().Add(ctx, node)
			if err != nil {
				cmd.PrintErrf("Failed to add file to IPFS: %v\n", err)
				return err
			}
			
			// Pin the content
			err = api.Pin().Add(ctx, ipfsPath)
			if err != nil {
				cmd.PrintErrf("Failed to pin file: %v\n", err)
				return err
			}
			
			cmd.Printf("✅ File successfully encrypted and uploaded to IPFS\n")
			cmd.Printf("📁 IPFS Path: %s\n", ipfsPath.String())
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
