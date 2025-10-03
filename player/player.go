package player

import (
	"fmt"
	"strings"
	"time"

	"github.com/qusicapp/qusic/logger"
	"github.com/qusicapp/qusic/lyrics"
	"github.com/qusicapp/qusic/streamer"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	yt "github.com/kkdai/youtube/v2"
)

type Named struct {
	Name, URL, ID string
}

type Artist struct {
	Named
	Thumbnails Thumbnails
}

type Album struct {
	Named
	Thumbnails Thumbnails
	Year       int
}

type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type SearchResult struct {
	TopResult Song
	Songs     []Song
	Artists   []Artist
	Albums    []Album
}

type Thumbnails []Thumbnail

const (
	YTMusic = 1
)

func (t Thumbnails) Min() Thumbnail {
	i := -1
	x, y := 0, 0
	for in, thumbnail := range t {
		if in == 0 {
			x, y = thumbnail.Width, thumbnail.Height
			i = 0
		}
		if thumbnail.Width < x && thumbnail.Height < y {
			x, y = thumbnail.Width, thumbnail.Height
			i = in
		}
	}
	if i == -1 {
		return Thumbnail{}
	}
	return t[i]
}

func (t Thumbnails) Max() Thumbnail {
	i := -1
	x, y := 0, 0
	for in, thumbnail := range t {
		if thumbnail.Width > x && thumbnail.Height > y {
			x, y = thumbnail.Width, thumbnail.Height
			i = in
		}
	}
	if i == -1 {
		return Thumbnail{}
	}
	return t[i]
}

type Song struct {
	Video  *yt.Video
	Format beep.Format

	Provider int

	Lyrics lyrics.Song

	Name, URL, ID string
	Album         Album
	Artists       []Artist
	Thumbnails    Thumbnails

	Duration time.Duration
	Plays    int
}

type Source interface {
	GetVideo(*Song)
	Search(query string) SearchResult
}

type Player struct {
	Source
	queue []*Song

	paused      bool
	currentSong int

	streamer *streamer.Streamer

	SongFinished chan struct{}
}

func New(src Source) *Player {
	st := streamer.NewStreamer()
	p := &Player{
		queue:        make([]*Song, 0),
		Source:       src,
		streamer:     st,
		currentSong:  -1,
		SongFinished: make(chan struct{}),
	}

	speaker.Init(streamer.SampleRate, streamer.SampleRate.N(time.Second/10))

	return p
}

func (p *Player) SetSpeed(x float64) {
	p.streamer.SetRatio(x)
}

func (p *Player) Volume() float64 {
	return p.streamer.Volume()
}

func (p *Player) SetVolume(v float64) {
	p.streamer.SetVolume(v)
}

func (p *Player) SetMute(b bool) {
	p.streamer.SetMute(b)
}

func (p *Player) Mute() bool {
	return p.streamer.Mute()
}

func (p *Player) Queue() []*Song {
	return p.queue
}

func (p *Player) CurrentIndex() int {
	return p.currentSong
}

func (p *Player) AddToQueue(song *Song) {
	p.queue = append(p.queue, song)
}

func (p *Player) PauseCycle() error {
	p.paused = !p.paused
	p.streamer.SetPaused(p.paused)
	return nil
}

func (p *Player) Paused() bool {
	return p.paused
}

func (p *Player) CurrentSong() *Song {
	if p.currentSong < 0 || p.currentSong >= len(p.queue) {
		return nil
	}
	return p.queue[p.currentSong]
}

func (p *Player) Playing() bool {
	return p.currentSong != -1 && p.streamer.Playing()
}

func (p *Player) postodur(pos int) time.Duration {
	return streamer.SampleRate.D(pos)
}

func (p *Player) durtopos(dur time.Duration) int {
	n := streamer.SampleRate.N(dur)
	return n
}

// Returns the current time position in the song
func (p *Player) TimePosition() time.Duration {
	return p.postodur(p.streamer.Position())
}

// Returns the time remaining for the song
func (p *Player) TimeRemaining() time.Duration {
	pos := p.TimePosition()
	l := p.postodur(p.streamer.Len())
	return l - pos
}

func (p *Player) SeekRaw(pos int) error {
	return p.streamer.Seek(pos)
}

func (p *Player) Seek(dur time.Duration) error {
	return p.SeekRaw(p.durtopos(dur))
}

func (p *Player) ClearQueue() {
	p.queue = make([]*Song, 0)
	p.currentSong = -1
}

func (p *Player) PlayNow(s *Song) error {
	p.ClearQueue()
	p.AddToQueue(s)
	if p.paused {
		if err := p.PauseCycle(); err != nil {
			return err
		}
	}
	return p.Play(0)
}

func (p *Player) SetCurrentSong(i int) {
	if i >= 0 && i < len(p.queue) {
		p.currentSong = i
	}
}

func (p *Player) Play(i int) error {
	if i < 0 || i >= len(p.queue) {
		return fmt.Errorf("índice fuera de rango")
	}

	song := p.queue[i]

	// Forzar provider YTMusic
	song.Provider = YTMusic

	logger.Infof("Reproduciendo - Nombre: %s, Provider: YTMusic", song.Name)

	var (
		stream beep.StreamSeekCloser
		format beep.Format
		err    error
	)

	logger.Infof("Usando YTMusic provider para: %s", song.Name)

	if song.Video == nil {
		videoID := extractVideoID(song.URL)
		if videoID != "" {
			// Obtener el video REAL de YouTube
			client := yt.Client{}
			video, err := client.GetVideo(videoID)
			if err != nil {
				logger.Errorf("Error obteniendo video de YouTube: %v", err)
				return fmt.Errorf("error obteniendo video de YouTube: %v", err)
			}
			song.Video = video
			logger.Infof("Video de YouTube obtenido: %s", video.Title)

			// Mostrar formatos disponibles para debug
			logger.Infof("Formatos disponibles para %s:", videoID)
			for i, fmt := range video.Formats {
				logger.Infof("  [%d] MimeType: %s, Quality: %s, AudioChannels: %d",
					i, fmt.MimeType, fmt.Quality, fmt.AudioChannels)
			}
		}
	}

	if song.Video == nil {
		return fmt.Errorf("video es nil para canción YTMusic: %s", song.Name)
	}

	// Intentar Opus/WebM primero, luego fallback a otros formatos
	stream, format, err = streamer.NewYTWebMOpusStreamer(song.Video)
	if err != nil {
		logger.Infof("Opus/WebM falló: %v", err)

		// Fallback a AAC/MP4
		stream, format, err = p.tryAACFallback(song.Video)
		if err != nil {
			logger.Errorf("Todos los métodos de streaming fallaron: %v", err)
			return fmt.Errorf("no se encontró formato de audio compatible: %v", err)
		}
	}

	if err != nil {
		return err
	}

	p.currentSong = i
	song.Format = format

	speaker.Clear()
	p.streamer.SetStreamer(stream)
	speaker.Play(beep.Seq(p.streamer, beep.Callback(func() {
		p.SongFinished <- struct{}{}
	})))

	logger.Infof("Reproducción iniciada exitosamente para: %s", song.Name)
	return nil
}

// Función de fallback para AAC/MP4
func (p *Player) tryAACFallback(video *yt.Video) (beep.StreamSeekCloser, beep.Format, error) {
	logger.Infof("Intentando fallback AAC/MP4...")

	// Buscar formatos AAC/MP4
	for _, fmt := range video.Formats {
		if strings.Contains(fmt.MimeType, "audio/mp4") ||
			strings.Contains(fmt.MimeType, "audio/aac") ||
			strings.Contains(fmt.MimeType, "mp4a") {
			logger.Infof("Formato AAC/MP4 encontrado: %s", fmt.MimeType)

			// Aquí necesitarías implementar el streamer para AAC
			// Por ahora retornamos error
			break
		}
	}

	return nil, beep.Format{}, fmt.Errorf("fallback AAC no implementado")
}

// Función para extraer Video ID de URL de YouTube
func extractVideoID(url string) string {
	logger.Infof("Extrayendo video ID de: %s", url)

	// Patrones de URL de YouTube
	patterns := []string{
		"music.youtube.com/watch?v=",
		"youtube.com/watch?v=",
		"youtu.be/",
		"www.youtube.com/watch?v=",
	}

	for _, pattern := range patterns {
		if strings.Contains(url, pattern) {
			parts := strings.Split(url, pattern)
			if len(parts) > 1 {
				videoID := parts[1]
				// Limpiar parámetros adicionales
				videoID = strings.Split(videoID, "&")[0]
				videoID = strings.Split(videoID, "?")[0]
				videoID = strings.Split(videoID, "#")[0]
				videoID = strings.TrimSpace(videoID)

				logger.Infof("Video ID extraído: %s (patrón: %s)", videoID, pattern)
				return videoID
			}
		}
	}

	logger.Infof("No se pudo extraer video ID de URL: %s", url)
	return ""
}

// Next avanza a la siguiente canción
func (p *Player) Next() error {
	if p.currentSong+1 >= len(p.queue) {
		return fmt.Errorf("no hay más canciones en la cola")
	}

	// Detener reproducción actual
	speaker.Clear()

	// Reproducir siguiente canción
	return p.Play(p.currentSong + 1)
}

// Previous vuelve a la canción anterior
func (p *Player) Previous() error {
	if p.currentSong-1 < 0 {
		return fmt.Errorf("no hay canción anterior")
	}

	// Detener reproducción actual
	speaker.Clear()

	// Reproducir canción anterior
	return p.Play(p.currentSong - 1)
}

// RemoveFromQueue elimina una canción de la cola
func (p *Player) RemoveFromQueue(index int) error {
	if index < 0 || index >= len(p.queue) {
		return fmt.Errorf("índice fuera de rango")
	}

	p.queue = append(p.queue[:index], p.queue[index+1:]...)

	// Ajustar currentSong si es necesario
	if p.currentSong >= index && p.currentSong > 0 {
		p.currentSong--
	}

	return nil
}
func (p *Player) Stop() {
	speaker.Clear()
	p.currentSong = -1
}

func (p *Player) Close() {
	p.streamer.Close()
	speaker.Close()
}
