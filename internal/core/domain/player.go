package domain

import (
	goutubedl "github.com/wader/goutubedl"
)

type MusicProvider interface {
	Search(query string) ([]Song, error)
	GetStream(song Song) (goutubedl.Result, error) // Cambiado a goutubedl.Result
}

type Player interface {
	Play(stream goutubedl.Result) error // Cambiado a goutubedl.Result
	Stop() error
	IsPlaying() bool
}

type PlaylistRepository interface {
	Add(song Song) error
	Remove(requester string) error
	GetAll() ([]Song, error)
	GetCurrentIndex() int
	SetCurrentIndex(index int)
	IsPlaying() bool
	SetPlaying(playing bool)
}
