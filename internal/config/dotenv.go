package config

import (
	"sync"

	"github.com/joho/godotenv"
)

var loadOnce sync.Once

// LoadEnv loads a .env file from the current working dir.
func LoadEnv() {
	loadOnce.Do(func() {
		if err := godotenv.Load(); err != nil {
			// .env not found or not readable -> keep OS env and continue
			return
		}
	})
}
