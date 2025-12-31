package main

import (
	"embed"
	"os"

	"BMongo-VIP/internal/crypto"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

var compiledPasswordHash string
var compiledEnv map[string]string // Mapa para injetar .env via secrets_gen.go

var envEncContent string

func main() {
	godotenv.Load()

	// Injeção direta via build.ps1 (secrets_gen.go)
	for key, value := range compiledEnv {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	if envEncContent != "" {
		decrypted, err := crypto.Decrypt(envEncContent, crypto.Key)
		if err == nil {
			embeddedEnv, _ := godotenv.Unmarshal(decrypted)
			for key, value := range embeddedEnv {
				if os.Getenv(key) == "" {
					os.Setenv(key, value)
				}
			}
		}
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Digisat Tools Suite",
		Width:  1100,
		Height: 850,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 22, B: 35, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
