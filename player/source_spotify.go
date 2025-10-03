package player

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/qusicapp/qusic/logger"
	"github.com/qusicapp/qusic/preferences"
	"github.com/qusicapp/qusic/spotify"
	"github.com/qusicapp/qusic/youtube"

	yt "github.com/kkdai/youtube/v2"
)

var ytclient = yt.Client{
	HTTPClient: &http.Client{
		Timeout: 30 * time.Second,
	},
}

func init() {
	ytclient.HTTPClient.Transport = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	logger.Info("YouTube client initialized with custom configuration")
}

func NewSpotifySource(client *spotify.Client) SpotifySource {
	return SpotifySource{client: client}
}

type SpotifySource struct {
	client *spotify.Client
}

func (s SpotifySource) Client() *spotify.Client {
	return s.client
}

func (source SpotifySource) GetVideo(s *Song) {
	logger.Infof("GetVideo called for: %s by %s", s.Name, s.Artists[0].Name)
	logger.Infof("Conditions - spotify.download_yt: %t, cookie_empty: %t",
		preferences.Preferences.Bool("spotify.download_yt"),
		source.client.Cookie_sp_dc == "")

	if preferences.Preferences.Bool("spotify.download_yt") || source.client.Cookie_sp_dc == "" {
		// 1. Manejar errores y resultados vacíos
		searchQuery := s.Artists[0].Name + " - " + s.Name
		logger.Infof("Searching YouTube for: %s", searchQuery)

		v, err := (*youtube.MusicClient).SearchSongs(nil, searchQuery)
		if err != nil {
			logger.Errorf("Error searching YouTube for %s: %v", s.Name, err)
			return
		}

		if len(v) == 0 {
			logger.Infof("No YouTube results found for: %s", s.Name)
			return
		}

		logger.Infof("Found %d YouTube results for: %s", len(v), s.Name)

		var vid *youtube.Video

		// 2. Relajar la lógica de búsqueda
		for i, video := range v {
			durationDiff := abs(video.Duration - s.Duration)
			logger.Infof("Result %d: %s (duration: %s, diff: %s)",
				i+1, video.Title, video.Duration, durationDiff)

			// Primero intentar match exacto de duración
			if durationDiff <= 10*time.Second {
				logger.Infof("Found duration match for %s: %s", s.Name, video.Title)
				vid = &v[i]
				break
			}
		}

		// 3. Si no encontró por duración, usar el primer resultado
		if vid == nil {
			logger.Infof("No exact duration match for %s, using first result: %s", s.Name, v[0].Title)
			vid = &v[0]
		}

		// 4. Obtener los detalles del video
		logger.Infof("Getting video details for ID: %s", vid.VideoID)
		video, err := ytclient.GetVideo(vid.VideoID)
		if err != nil {
			logger.Errorf("Error getting video details for %s: %v", vid.VideoID, err)
			return
		}

		s.Video = video
		logger.Infof("Successfully got video for %s: %s", s.Name, vid.VideoID)
	} else {
		logger.Infof("Using Spotify streaming for: %s", s.Name)
	}
}

func (source SpotifySource) Search(query string) SearchResult {
	var result SearchResult
	res, _ := source.client.Search(query, spotify.QueryAll, "", nil, nil, false)
	if len(res.Tracks.Items) == 0 {
		goto csongs
	}
	result.TopResult = source.Song(res.Tracks.Items[0])

	result.Songs = make([]Song, len(res.Tracks.Items))
	for i, song := range res.Tracks.Items {
		result.Songs[i] = source.Song(song)
	}

csongs:
	result.Artists = make([]Artist, len(res.Artists.Items))
	for i, artist := range res.Artists.Items {
		result.Artists[i] = source.Artist(artist)
	}

	return result
}

func (source SpotifySource) Song(a spotify.TrackObject) Song {
	var s Song
	s.Provider = Spotify
	s.Album = source.Album(a.Album)
	s.Artists = make([]Artist, len(a.Artists))
	for i, a := range a.Artists {
		s.Artists[i] = source.Artist(a)
	}
	s.Name = a.Name
	s.URL = a.ExternalURLs.Spotify
	s.Thumbnails = *(*Thumbnails)(unsafe.Pointer(&a.Album.Images))
	s.Duration = time.Duration(a.DurationMS) * time.Millisecond
	s.ID = a.ID

	return s
}

func (source SpotifySource) Artist(a spotify.ArtistObject) Artist {
	var artist Artist
	artist.Name = a.Name
	artist.ID = a.ID
	artist.URL = a.ExternalURLs.Spotify
	artist.Listeners = a.Followers.Total
	artist.Thumbnails = *(*Thumbnails)(unsafe.Pointer(&a.Images))

	return artist
}

func (source SpotifySource) Album(a spotify.SimplifiedAlbumObject) Album {
	var album Album
	album.Name = a.Name
	album.ID = a.ID
	album.URL = a.ExternalURLs.Spotify
	album.Year, _ = strconv.Atoi(strings.Split(a.ReleaseDate, "-")[0])
	album.Thumbnails = *(*Thumbnails)(unsafe.Pointer(&a.Images))

	return album
}
