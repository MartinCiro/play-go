package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/MartinCiro/play-go/internal/core/application/ports"
	"github.com/MartinCiro/play-go/internal/core/application/service"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/persistence/memory"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/player/ffplay"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/providers/youtube"
	"github.com/MartinCiro/play-go/pkg/ffmpeg"
	"github.com/MartinCiro/play-go/pkg/logger"
)

func main() {
	fmt.Println("🎵 Bot de Música Simplificado")
	fmt.Println("==============================")
	fmt.Println("Comandos disponibles:")
	fmt.Println("!play [canción] - Añadir canción a la cola")
	fmt.Println("!revoke - Revocar tu última canción")
	fmt.Println("!skip - Saltar canción actual")
	fmt.Println("!queue - Mostrar cola actual")
	fmt.Println("!exit - Salir del programa")

	// Verificar que ffplay está disponible
	if err := ffmpeg.CheckOrInstall(); err != nil {
		logger.Fatalf("❌ No se pudo instalar ffplay: %v", err)
	}

	// Inicializar dependencias
	ytProvider := youtube.NewYouTubeProvider()
	ffPlayer := ffplay.NewFFPlayPlayer()
	playlistRepo := memory.NewPlaylistRepository()

	// Crear servicio
	musicService := service.NewMusicService(ports.ServiceDependencies{
		Provider: ytProvider,
		Player:   ffPlayer,
		Repo:     playlistRepo,
	})

	// Iniciar CLI
	runCLI(musicService)
}

func runCLI(service ports.MusicService) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(input)

		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])

		switch command {
		case "!play":
			if len(parts) < 2 {
				fmt.Println("❌ Uso: !play [nombre de la canción]")
				continue
			}
			songName := strings.Join(parts[1:], " ")
			if err := service.PlaySong(songName, "Usuario"); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "!revoke":
			if err := service.RevokeSong("Usuario"); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "!skip":
			if err := service.Skip(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				fmt.Println("✅ Saltando canción actual...")
			}

		case "!queue":
			service.ShowQueue()

		case "!exit":
			fmt.Println("👋 ¡Hasta luego!")
			return

		default:
			fmt.Println("❌ Comando no reconocido. Comandos: !play, !revoke, !skip, !queue, !exit")
		}
	}
}
