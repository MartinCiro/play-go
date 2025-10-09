package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MartinCiro/play-go/internal/core/application/ports"
	"github.com/MartinCiro/play-go/internal/core/domain"
	"github.com/MartinCiro/play-go/pkg/logger"
)

type MusicService struct {
	provider domain.MusicProvider
	player   domain.Player
	repo     domain.PlaylistRepository
	mu       sync.Mutex
}

func NewMusicService(deps ports.ServiceDependencies) ports.MusicService {
	return &MusicService{
		provider: deps.Provider,
		player:   deps.Player,
		repo:     deps.Repo,
	}
}

func (ms *MusicService) RevokeSong(requester string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	songs, _ := ms.repo.GetAll()
	currentIndex := ms.repo.GetCurrentIndex()

	// Buscar la última canción del solicitante
	for i := len(songs) - 1; i >= 0; i-- {
		if songs[i].Requester == requester {
			logger.Infof("🗑️ Revocando canción: %s (solicitante: %s)", songs[i].Title, requester)

			// Si es la canción actualmente reproduciéndose, detenerla
			if i == currentIndex {
				logger.Info("⏹️ Deteniendo reproducción actual (canción revocada)")
				if err := ms.player.Stop(); err != nil {
					logger.Errorf("❌ Error deteniendo reproducción: %v", err)
				}
				// Reiniciar el índice actual
				ms.repo.SetCurrentIndex(-1)
				ms.repo.SetPlaying(false)
			}

			// Remover de la playlist
			if err := ms.repo.Remove(requester); err != nil {
				return err
			}

			// Si se removió una canción y hay más en la lista, reiniciar reproducción
			if ms.repo.IsPlaying() && currentIndex >= len(songs)-1 {
				logger.Info("🔄 Reiniciando reproducción después de revocar canción actual")
				go ms.startPlayback()
			}

			return nil
		}
	}

	return fmt.Errorf("❌ No se encontraron canciones solicitadas por %s", requester)
}

func (ms *MusicService) PlaySong(songName string, requester string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	logger.Infof("🎵 Buscando: %s...", songName)

	songs, err := ms.provider.Search(songName)
	if err != nil {
		logger.Errorf("❌ Error buscando canción: %v", err)
		return fmt.Errorf("error al buscar la canción: %v", err)
	}

	if len(songs) == 0 {
		logger.Warnf("❌ No se encontraron resultados para: %s", songName)
		return fmt.Errorf("no se encontraron resultados para: %s", songName)
	}

	song := songs[0]
	song.Requester = requester

	// Verificar si ya está en la playlist
	existingSongs, _ := ms.repo.GetAll()
	for _, existing := range existingSongs {
		if existing.ID == song.ID {
			logger.Warnf("❌ La canción ya está en la playlist: %s", song.Title)
			return fmt.Errorf("❌ La canción ya está en la playlist")
		}
	}

	if err := ms.repo.Add(song); err != nil {
		logger.Errorf("❌ Error añadiendo a playlist: %v", err)
		return err
	}

	logger.Infof("✅ Añadido: %s (Solicitado por: %s)", song.Title, requester)

	// Si no hay nada reproduciéndose, iniciar reproducción
	if !ms.repo.IsPlaying() {
		logger.Info("🚀 Iniciando reproducción en background...")
		go ms.startPlayback()
	}

	return nil
}

func (ms *MusicService) Skip() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.player.IsPlaying() {
		logger.Warn("❌ No hay ninguna canción reproduciéndose")
		return fmt.Errorf("❌ No hay ninguna canción reproduciéndose")
	}

	logger.Info("⏭️  Saltando canción actual...")
	return ms.player.Stop()
}

func (ms *MusicService) ShowQueue() {
	songs, _ := ms.repo.GetAll()
	currentIndex := ms.repo.GetCurrentIndex()

	logger.Info("\n🎵 Cola de Reproducción:")
	if len(songs) == 0 {
		logger.Info("   La cola está vacía")
		return
	}

	// Construir la cola como string
	var queue strings.Builder
	for i, song := range songs {
		status := "  "
		if i == currentIndex {
			status = "▶️"
		}
		queue.WriteString(fmt.Sprintf("   %s %d. %s\n", status, i+1, song.Title))
		queue.WriteString(fmt.Sprintf("      👤 %s\n", song.Requester))
	}
	queue.WriteString(fmt.Sprintf("\n   Total: %d canciones en cola\n", len(songs)))

	logger.Info(queue.String())
}

func (ms *MusicService) startPlayback() {
	ms.repo.SetPlaying(true)
	defer ms.repo.SetPlaying(false)

	logger.Info("🎶 Iniciando reproducción de playlist...")

	for {
		songs, _ := ms.repo.GetAll()
		currentIndex := ms.repo.GetCurrentIndex()

		if currentIndex >= len(songs)-1 {
			logger.Info("📭 Fin de la playlist alcanzado")
			break
		}

		nextIndex := currentIndex + 1
		ms.repo.SetCurrentIndex(nextIndex)
		currentSong := songs[nextIndex]

		logger.Infof("\n🎵 Reproduciendo (%d/%d): %s", nextIndex+1, len(songs), currentSong.Title)
		logger.Infof("   👤 Solicitado por: %s", currentSong.Requester)

		stream, err := ms.provider.GetStream(currentSong)
		if err != nil {
			logger.Errorf("❌ Error obteniendo stream: %s: %v", currentSong.Title, err)
			continue
		}

		if err := ms.player.Play(stream); err != nil {
			logger.Errorf("❌ Error reproduciendo %s: %v", currentSong.Title, err)
			continue
		}

		// Esperar hasta que la canción termine o sea skipeada
		for ms.player.IsPlaying() {
			time.Sleep(500 * time.Millisecond)
		}

		logger.Infof("✅ Completado: %s", currentSong.Title)
	}

	logger.Info("\n🎉 ¡Playlist completada!")
	ms.repo.SetCurrentIndex(-1)
}
