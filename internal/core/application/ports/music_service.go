package ports

import "github.com/MartinCiro/play-go/internal/core/domain"

// MusicService define el puerto para el servicio principal de música
type MusicService interface {
	PlaySong(songName string, requester string) error
	RevokeSong(requester string) error
	Skip() error
	ShowQueue()
}

// ServiceDependencies contiene las dependencias necesarias para el servicio
type ServiceDependencies struct {
	Provider domain.MusicProvider
	Player   domain.Player
	Repo     domain.PlaylistRepository
}
