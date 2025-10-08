package domain

import (
	"io"
)

// MusicProvider define la interfaz para buscar y obtener streams de música
type MusicProvider interface {
	Search(query string) ([]Song, error)
	GetStream(song Song) (io.ReadCloser, error)
}

// Player define la interfaz para reproducir audio
type Player interface {
	Play(stream io.ReadCloser) error
	Stop() error
	IsPlaying() bool
}

// PlaylistRepository define la interfaz para gestionar la playlist
type PlaylistRepository interface {
	Add(song Song) error
	Remove(requester string) error
	GetAll() ([]Song, error)
	GetCurrentIndex() int
	SetCurrentIndex(index int)
	IsPlaying() bool
	SetPlaying(playing bool)
}
