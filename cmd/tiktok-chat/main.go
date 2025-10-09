package main

import (
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/MartinCiro/play-go/internal/core/application/ports"
	"github.com/MartinCiro/play-go/internal/core/application/service"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/persistence/memory"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/player/ffplay"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/providers/youtube"
	"github.com/MartinCiro/play-go/pkg/ffmpeg"
	"github.com/MartinCiro/play-go/pkg/logger"
	"github.com/steampoweredtaco/gotiktoklive"
)

func main() {
	if len(os.Args) < 2 {
		logger.Fatal("Uso: go run main.go <username_tiktok>")
	}

	username := os.Args[1]
	logger.Infof("Conectando al livestream de: @%s", username)

	// Inicializar el bot de música
	musicService := initializeMusicBot()
	if musicService == nil {
		logger.Fatal("❌ No se pudo inicializar el bot de música")
	}

	logger.Info("✅ Bot de música inicializado")

	// Crear instancia de TikTok
	tiktok, err := gotiktoklive.NewTikTok()
	if err != nil {
		logger.Fatalf("Error creando cliente TikTok: %v", err)
	}

	// Trackear usuario por username
	live, err := tiktok.TrackUser(username)
	if err != nil {
		logger.Fatalf("Error conectando al stream: %v", err)
	}

	logger.Info("Conectado exitosamente!")
	logger.Info("==================================================")
	logger.Info("🎵 Comandos disponibles en el chat:")
	logger.Info("!play [canción] - Añadir canción a la playlist")
	logger.Info("!skip - Saltar canción actual")
	logger.Info("!queue - Mostrar cola de reproducción")
	logger.Info("!revoke - Eliminar tu última canción")
	logger.Info("==================================================")
	logger.Info("Escuchando comandos del chat... (Ctrl+C para salir)")

	// Recibir eventos del livestream
	go func() {
		for event := range live.Events {
			// Usar reflexión para acceder a los campos
			eventValue := reflect.ValueOf(event)
			if eventValue.Kind() == reflect.Ptr {
				eventValue = eventValue.Elem()
			}

			// Buscar el campo Comment
			commentField := eventValue.FieldByName("Comment")
			if commentField.IsValid() && commentField.Kind() == reflect.String && commentField.String() != "" {
				comment := commentField.String()

				// Buscar el campo User
				userField := eventValue.FieldByName("User")
				username := "Anónimo"
				if userField.IsValid() {
					username = extractUsername(userField)
				}

				// Mostrar comentario
				logger.Infof("💬 @%s: %s", username, comment)

				// Procesar comandos
				processCommand(comment, username, musicService)
			}
		}
	}()

	// Esperar señal para terminar
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Saliendo...")
}

func initializeMusicBot() ports.MusicService {
	// Verificar e instalar ffplay si es necesario
	if err := ffmpeg.CheckOrInstall(); err != nil {
		logger.Errorf("❌ No se pudo instalar ffplay: %v", err)
		return nil
	}

	// Inicializar componentes del bot de música
	ytProvider := youtube.NewYouTubeProvider()
	ffPlayer := ffplay.NewFFPlayPlayer()
	playlistRepo := memory.NewPlaylistRepository()

	return service.NewMusicService(ports.ServiceDependencies{
		Provider: ytProvider,
		Player:   ffPlayer,
		Repo:     playlistRepo,
	})
}

func processCommand(comment, username string, musicService ports.MusicService) {
	comment = strings.TrimSpace(comment)

	// Solo procesar comandos que empiecen con !
	if !strings.HasPrefix(comment, "!") {
		return
	}

	parts := strings.Fields(comment)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "!play":
		if len(parts) < 2 {
			logger.Warnf("⚠️ @%s: Uso: !play [nombre de la canción]", username)
			return
		}
		songName := strings.Join(parts[1:], " ")
		logger.Infof("🎵 @%s solicitó: %s", username, songName)
		if err := musicService.PlaySong(songName, "@"+username); err != nil {
			logger.Errorf("❌ Error con @%s: %v", username, err)
		}

	case "!skip":
		logger.Infof("⏭️ @%s solicitó saltar canción", username)
		if err := musicService.Skip(); err != nil {
			logger.Errorf("❌ Error con @%s: %v", username, err)
		} else {
			logger.Infof("✅ Saltando canción (solicitado por @%s)", username)
		}

	case "!queue":
		logger.Infof("📋 @%s solicitó ver la cola", username)
		musicService.ShowQueue()

	case "!revoke":
		logger.Infof("🗑️ @%s solicitó revocar su canción", username)
		if err := musicService.RevokeSong("@" + username); err != nil {
			logger.Errorf("❌ Error con @%s: %v", username, err)
		}

	default:
		// Comando no reconocido
		logger.Warnf("❓ @%s usó comando no reconocido: %s", username, command)
	}
}

func extractUsername(userValue reflect.Value) string {
	if userValue.Kind() == reflect.Ptr {
		if userValue.IsNil() {
			return ""
		}
		userValue = userValue.Elem()
	}

	if userValue.Kind() == reflect.Struct {
		// Intentar campos comunes para username
		possibleFields := []string{"Username", "Nickname", "DisplayName", "UniqueID", "Name"}

		for _, fieldName := range possibleFields {
			field := userValue.FieldByName(fieldName)
			if field.IsValid() && field.Kind() == reflect.String && field.String() != "" {
				return field.String()
			}
		}

		// Si no encontramos campos específicos, explorar todos los campos string
		for i := 0; i < userValue.NumField(); i++ {
			field := userValue.Field(i)
			if field.Kind() == reflect.String && field.String() != "" {
				fieldName := userValue.Type().Field(i).Name
				// Ignorar campos que probablemente no sean usernames
				if fieldName != "ID" && fieldName != "Email" && fieldName != "Avatar" {
					return field.String()
				}
			}
		}
	}

	return ""
}
