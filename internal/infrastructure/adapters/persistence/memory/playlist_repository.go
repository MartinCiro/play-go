package memory

import (
	"fmt"
	"sync"

	"github.com/MartinCiro/play-go/internal/core/domain"
)

type PlaylistRepository struct {
	playlist     []domain.Song
	currentIndex int
	isPlaying    bool
	mu           sync.Mutex
}

func NewPlaylistRepository() *PlaylistRepository {
	return &PlaylistRepository{
		playlist:     []domain.Song{},
		currentIndex: -1,
		isPlaying:    false,
	}
}

func (pr *PlaylistRepository) Add(song domain.Song) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.playlist = append(pr.playlist, song)
	return nil
}

func (pr *PlaylistRepository) ReplaceAll(newPlaylist []domain.Song) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.playlist = newPlaylist
	// Resetear índice actual ya que la lista cambió
	pr.currentIndex = -1
}

func (pr *PlaylistRepository) Remove(requester string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(pr.playlist) == 0 {
		return fmt.Errorf("❌ La playlist está vacía")
	}

	// Buscar la última canción del solicitante
	for i := len(pr.playlist) - 1; i >= 0; i-- {
		if pr.playlist[i].Requester == requester {
			removedSong := pr.playlist[i].Title

			// Si es la canción actual, ajustar el índice
			if i == pr.currentIndex {
				pr.currentIndex = -1
			} else if i < pr.currentIndex {
				pr.currentIndex--
			}

			// Remover de la playlist
			pr.playlist = append(pr.playlist[:i], pr.playlist[i+1:]...)

			fmt.Printf("✅ Canción revocada: %s (Solicitante: %s)\n", removedSong, requester)
			return nil
		}
	}

	return fmt.Errorf("❌ No se encontraron canciones solicitadas por %s", requester)
}

func (pr *PlaylistRepository) GetAll() ([]domain.Song, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.playlist, nil
}

func (pr *PlaylistRepository) GetCurrentIndex() int {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.currentIndex
}

func (pr *PlaylistRepository) SetCurrentIndex(index int) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.currentIndex = index
}

func (pr *PlaylistRepository) IsPlaying() bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.isPlaying
}

func (pr *PlaylistRepository) SetPlaying(playing bool) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.isPlaying = playing
}
