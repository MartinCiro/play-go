package youtube

import (
	"context"
	"fmt"

	"github.com/MartinCiro/play-go/internal/core/domain"
	goutubedl "github.com/wader/goutubedl"
)

type YouTubeProvider struct{}

func NewYouTubeProvider() *YouTubeProvider {
	return &YouTubeProvider{}
}

func (y *YouTubeProvider) Search(query string) ([]domain.Song, error) {
	searchQuery := "ytsearch1:" + query

	result, err := goutubedl.New(context.Background(), searchQuery, goutubedl.Options{})
	if err != nil {
		return nil, fmt.Errorf("error al buscar la canción: %v", err)
	}

	var songs []domain.Song

	if len(result.Info.Entries) > 0 {
		// Es una playlist de resultados de búsqueda
		for _, entry := range result.Info.Entries {
			songs = append(songs, domain.Song{
				ID:    entry.ID,
				Title: entry.Title,
				URL:   entry.WebpageURL,
			})
		}
	} else {
		// Resultado directo
		songs = append(songs, domain.Song{
			ID:    result.Info.ID,
			Title: result.Info.Title,
			URL:   result.Info.WebpageURL,
		})
	}

	return songs, nil
}

func (y *YouTubeProvider) GetStream(song domain.Song) (goutubedl.Result, error) {
	result, err := goutubedl.New(context.Background(), song.URL, goutubedl.Options{})
	if err != nil {
		return goutubedl.Result{}, fmt.Errorf("error al obtener video: %v", err)
	}

	// Devolvemos el Result directamente para mantener compatibilidad con tu código actual
	return result, nil
}
