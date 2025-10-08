package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/MartinCiro/play-go/internal/core/application/ports"
	"github.com/MartinCiro/play-go/internal/core/domain"
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

func (ms *MusicService) PlaySong(songName string, requester string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	fmt.Printf("🎵 Buscando: %s...\n", songName)

	songs, err := ms.provider.Search(songName)
	if err != nil {
		return fmt.Errorf("error al buscar la canción: %v", err)
	}

	if len(songs) == 0 {
		return fmt.Errorf("no se encontraron resultados para: %s", songName)
	}

	song := songs[0]
	song.Requester = requester

	// Verificar si ya está en la playlist
	existingSongs, _ := ms.repo.GetAll()
	for _, existing := range existingSongs {
		if existing.ID == song.ID {
			return fmt.Errorf("❌ La canción ya está en la playlist")
		}
	}

	if err := ms.repo.Add(song); err != nil {
		return err
	}

	fmt.Printf("✅ Añadido: %s (Solicitado por: %s)\n", song.Title, requester)

	// Si no hay nada reproduciéndose, iniciar reproducción
	if !ms.repo.IsPlaying() {
		go ms.startPlayback()
	}

	return nil
}

func (ms *MusicService) RevokeSong(requester string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return ms.repo.Remove(requester)
}

func (ms *MusicService) Skip() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.player.IsPlaying() {
		return fmt.Errorf("❌ No hay ninguna canción reproduciéndose")
	}

	return ms.player.Stop()
}

func (ms *MusicService) ShowQueue() {
	songs, _ := ms.repo.GetAll()
	currentIndex := ms.repo.GetCurrentIndex()

	fmt.Println("\n🎵 Cola de Reproducción:")
	if len(songs) == 0 {
		fmt.Println("   La cola está vacía")
		return
	}

	for i, song := range songs {
		status := "  "
		if i == currentIndex {
			status = "▶️"
		}
		fmt.Printf("   %s %d. %s\n", status, i+1, song.Title)
		fmt.Printf("      👤 %s\n", song.Requester)
	}
	fmt.Printf("\n   Total: %d canciones en cola\n", len(songs))
}

func (ms *MusicService) startPlayback() {
	ms.repo.SetPlaying(true)
	defer ms.repo.SetPlaying(false)

	for {
		songs, _ := ms.repo.GetAll()
		currentIndex := ms.repo.GetCurrentIndex()

		if currentIndex >= len(songs)-1 {
			break
		}

		nextIndex := currentIndex + 1
		ms.repo.SetCurrentIndex(nextIndex)
		currentSong := songs[nextIndex]

		fmt.Printf("\n🎵 Reproduciendo (%d/%d): %s\n", nextIndex+1, len(songs), currentSong.Title)
		fmt.Printf("   👤 Solicitado por: %s\n", currentSong.Requester)

		stream, err := ms.provider.GetStream(currentSong)
		if err != nil {
			fmt.Printf("❌ Error obteniendo stream: %s: %v\n", currentSong.Title, err)
			continue
		}

		// Play ahora es no bloqueante, necesitamos esperar de otra forma
		if err := ms.player.Play(stream); err != nil {
			fmt.Printf("❌ Error reproduciendo %s: %v\n", currentSong.Title, err)
			continue
		}

		// Esperar hasta que la canción termine o sea skipeada
		for ms.player.IsPlaying() {
			time.Sleep(500 * time.Millisecond)
		}

		fmt.Printf("✅ Completado: %s\n", currentSong.Title)
	}

	fmt.Println("\n🎉 ¡Playlist completada!")
	ms.repo.SetCurrentIndex(-1)
}
