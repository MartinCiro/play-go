package tiktok

import (
	"strings"

	"github.com/MartinCiro/play-go/internal/core/application/ports"
	"github.com/MartinCiro/play-go/pkg/logger"
)

type TikTokChat struct {
	musicService ports.MusicService
}

func NewTikTokChat(musicService ports.MusicService) *TikTokChat {
	return &TikTokChat{
		musicService: musicService,
	}
}

func (t *TikTokChat) HandleComment(username, comment string) {
	logger.Infof("💬 TikTok - @%s: %s", username, comment)

	// Parsear comandos
	command := t.parseCommand(comment)
	if command != nil {
		t.executeCommand(command, username)
	}
}

func (t *TikTokChat) parseCommand(comment string) *Command {
	comment = strings.TrimSpace(comment)

	if !strings.HasPrefix(comment, "!") {
		return nil
	}

	parts := strings.Fields(comment)
	if len(parts) == 0 {
		return nil
	}

	commandType := strings.ToLower(parts[0])

	switch commandType {
	case "!play":
		if len(parts) < 2 {
			return &Command{Type: "error", Data: "Uso: !play [canción]"}
		}
		songName := strings.Join(parts[1:], " ")
		return &Command{Type: "play", Data: songName}

	case "!skip":
		return &Command{Type: "skip", Data: ""}

	case "!queue":
		return &Command{Type: "queue", Data: ""}

	case "!revoke":
		return &Command{Type: "revoke", Data: ""}

	default:
		return nil
	}
}

func (t *TikTokChat) executeCommand(command *Command, username string) {
	switch command.Type {
	case "play":
		songName := command.Data.(string)
		logger.Infof("🎵 TikTok - @%s solicitó: %s", username, songName)
		if err := t.musicService.PlaySong(songName, "@"+username); err != nil {
			logger.Errorf("❌ Error en TikTok play: %v", err)
		}

	case "skip":
		logger.Infof("⏭️ TikTok - @%s solicitó skip", username)
		if err := t.musicService.Skip(); err != nil {
			logger.Errorf("❌ Error en TikTok skip: %v", err)
		}

	case "revoke":
		logger.Infof("🗑️ TikTok - @%s solicitó revoke", username)
		if err := t.musicService.RevokeSong("@" + username); err != nil {
			logger.Errorf("❌ Error en TikTok revoke: %v", err)
		}

	case "queue":
		logger.Infof("📋 TikTok - @%s solicitó queue", username)
		t.musicService.ShowQueue()

	case "error":
		logger.Warnf("⚠️ Comando inválido de @%s: %s", username, command.Data)
	}
}

type Command struct {
	Type string
	Data interface{}
}
