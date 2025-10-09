package main

import (
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/MartinCiro/play-go/internal/core/application/ports"
	"github.com/MartinCiro/play-go/internal/core/application/service"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/persistence/memory"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/player/ffplay"
	"github.com/MartinCiro/play-go/internal/infrastructure/adapters/providers/youtube"
	"github.com/MartinCiro/play-go/pkg/ffmpeg"
	"github.com/MartinCiro/play-go/pkg/logger"
	"github.com/steampoweredtaco/gotiktoklive"
)

const expirationHours = 8

// buildTime se establece durante la compilación con -ldflags
var buildTime string

var startTime time.Time

func main() {
	// Verificar expiración
	if isExpired() {
		showExpirationMessage()
		os.Exit(1)
	}

	startTime = time.Now() // Guardar tiempo de inicio

	if len(os.Args) < 2 {
		logger.Fatal("Uso: go run main.go <username_tiktok>")
	}

	username := os.Args[1]
	logger.Infof("Conectando al livestream de: @%s", username)
	logger.Infof("⏰ Inicio del bot: %s", startTime.Format("15:04:05"))

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
	logger.Info("Escuchando comandos NUEVOS del chat... (Ctrl+C para salir)")

	// Recibir eventos del livestream
	go func() {
		for event := range live.Events {
			// Filtrar solo mensajes nuevos (después del inicio del bot)
			if isNewMessage(event) {
				processTikTokEvent(event, musicService)
			}
		}
	}()

	// Esperar señal para terminar
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Saliendo...")
}

func isExpired() bool {
	bt := getBuildTime()
	expirationTime := bt.Add(time.Hour * expirationHours)

	// Mostrar información de tiempo
	logger.Infof("🕐 Build time: %s", bt.Format("2006-01-02 15:04:05"))
	logger.Infof("⏰ Expira: %s", expirationTime.Format("2006-01-02 15:04:05"))
	logger.Infof("⏱️ Tiempo restante: %v", time.Until(expirationTime).Round(time.Minute))

	return time.Now().After(expirationTime)
}

func getBuildTime() time.Time {
	if buildTime != "" {
		// Parsear el tiempo de compilación
		if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
			return t
		}
	}

	// Fallback: usar tiempo actual (para desarrollo sin -ldflags)
	return time.Now().Add(-1 * time.Hour)
}

func showExpirationMessage() {
	logger.Info("")
	logger.Info("🚫 =================================")
	logger.Info("🚫        VERSIÓN DE PRUEBA")
	logger.Info("🚫 =================================")
	logger.Infof("🚫 Esta versión ha expirado después de %d horas", expirationHours)
	logger.Info("🚫 ")
	logger.Info("💡 Para obtener la versión completa:")
	logger.Info("💡 • Contacta al desarrollador")
	logger.Info("💡 • Visita: https://github.com/MartinCiro/play-go")
	logger.Info("💡 • Email: soporte@playgo-app.com")
	logger.Info("")
}

// isNewMessage verifica si el mensaje es nuevo (posterior al inicio del bot)
func isNewMessage(event interface{}) bool {
	eventValue := reflect.ValueOf(event)
	if eventValue.Kind() == reflect.Ptr {
		eventValue = eventValue.Elem()
	}

	// Buscar campos de timestamp comunes
	timestampFields := []string{"Timestamp", "Time", "CreatedAt", "MsgID"}

	for _, fieldName := range timestampFields {
		field := eventValue.FieldByName(fieldName)
		if field.IsValid() {
			switch field.Kind() {
			case reflect.Int64, reflect.Int:
				// Si es un timestamp numérico (segundos o milisegundos)
				timestamp := field.Int()
				eventTime := time.Unix(timestamp, 0)
				return eventTime.After(startTime)
			case reflect.String:
				// Si es un string de timestamp, intentar parsear
				// (implementar según el formato que use la librería)
			case reflect.Struct:
				// Si es time.Time directamente
				if timeField, ok := field.Interface().(time.Time); ok {
					return timeField.After(startTime)
				}
			}
		}
	}

	// Si no podemos determinar el timestamp, asumir que es nuevo
	// pero mostrar advertencia
	logger.Warn("⚠️ No se pudo determinar timestamp del mensaje, procesando igual")
	return true
}

// processTikTokEvent procesa un evento de TikTok
func processTikTokEvent(event interface{}, musicService ports.MusicService) {
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

// Resto del código se mantiene igual...
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
