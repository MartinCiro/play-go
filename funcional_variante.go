package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	goutubedl "github.com/wader/goutubedl"
)

type VideoInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"webpage_url"`
	Requester string `json:"requester"`
}

type MusicPlayer struct {
	playlist       []VideoInfo
	currentProcess *os.Process
	currentIndex   int
	isPlaying      bool
	mu             sync.Mutex
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		playlist:     []VideoInfo{},
		currentIndex: -1,
		isPlaying:    false,
	}
}

// PlaySong busca y añade una canción a la cola
func (mp *MusicPlayer) PlaySong(songName string, requester string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	fmt.Printf("🎵 Buscando: %s...\n", songName)

	// Usar ytsearch para obtener el primer resultado
	searchQuery := "ytsearch1:" + songName

	result, err := goutubedl.New(context.Background(), searchQuery, goutubedl.Options{})
	if err != nil {
		return fmt.Errorf("error al buscar la canción: %v", err)
	}

	// Procesar el resultado de búsqueda
	var videoInfo VideoInfo

	if len(result.Info.Entries) > 0 {
		// Es una playlist de resultados de búsqueda
		firstResult := result.Info.Entries[0]
		videoInfo = VideoInfo{
			ID:        firstResult.ID,
			Title:     firstResult.Title,
			URL:       firstResult.WebpageURL,
			Requester: requester,
		}
	} else {
		// Resultado directo
		videoInfo = VideoInfo{
			ID:        result.Info.ID,
			Title:     result.Info.Title,
			URL:       result.Info.WebpageURL,
			Requester: requester,
		}
	}

	// Verificar si ya está en la playlist
	for _, v := range mp.playlist {
		if v.ID == videoInfo.ID {
			return fmt.Errorf("❌ La canción ya está en la playlist")
		}
	}

	mp.playlist = append(mp.playlist, videoInfo)
	fmt.Printf("✅ Añadido: %s (Solicitado por: %s)\n", videoInfo.Title, requester)

	// Si no hay nada reproduciéndose, iniciar reproducción
	if !mp.isPlaying {
		go mp.startPlayback()
	}

	return nil
}

// RevokeSong remueve la última canción solicitada por un usuario
func (mp *MusicPlayer) RevokeSong(requester string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if len(mp.playlist) == 0 {
		return fmt.Errorf("❌ La playlist está vacía")
	}

	// Buscar la última canción del solicitante
	for i := len(mp.playlist) - 1; i >= 0; i-- {
		if mp.playlist[i].Requester == requester {
			removedSong := mp.playlist[i].Title

			// Si es la canción actual, saltar a la siguiente
			if i == mp.currentIndex {
				mp.skipCurrent()
			}

			// Remover de la playlist
			mp.playlist = append(mp.playlist[:i], mp.playlist[i+1:]...)

			// Ajustar el índice actual si es necesario
			if i < mp.currentIndex {
				mp.currentIndex--
			}

			fmt.Printf("✅ Canción revocada: %s (Solicitante: %s)\n", removedSong, requester)
			return nil
		}
	}

	return fmt.Errorf("❌ No se encontraron canciones solicitadas por %s", requester)
}

// Skip salta la canción actual
func (mp *MusicPlayer) Skip() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if !mp.isPlaying {
		return fmt.Errorf("❌ No hay ninguna canción reproduciéndose")
	}

	return mp.skipCurrent()
}

// skipCurrent salta la canción actual (debe llamarse con el mutex bloqueado)
func (mp *MusicPlayer) skipCurrent() error {
	if mp.currentProcess != nil {
		// Detener el proceso actual
		exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", mp.currentProcess.Pid)).Run()
		mp.currentProcess = nil
	}
	return nil
}

// startPlayback inicia la reproducción de la playlist
func (mp *MusicPlayer) startPlayback() {
	mp.mu.Lock()
	mp.isPlaying = true
	mp.mu.Unlock()

	defer func() {
		mp.mu.Lock()
		mp.isPlaying = false
		mp.currentIndex = -1
		mp.mu.Unlock()
	}()

	for {
		mp.mu.Lock()
		if mp.currentIndex >= len(mp.playlist)-1 {
			mp.mu.Unlock()
			break
		}

		mp.currentIndex++
		currentSong := mp.playlist[mp.currentIndex]
		mp.mu.Unlock()

		fmt.Printf("\n🎵 Reproduciendo (%d/%d): %s\n", mp.currentIndex+1, len(mp.playlist), currentSong.Title)
		fmt.Printf("   👤 Solicitado por: %s\n", currentSong.Requester)

		if err := mp.playAudioDirect(currentSong); err != nil {
			fmt.Printf("❌ Error reproduciendo %s: %v\n", currentSong.Title, err)
			continue
		}

		fmt.Printf("✅ Completado: %s\n", currentSong.Title)
	}

	fmt.Println("\n🎉 ¡Playlist completada!")
}

func (mp *MusicPlayer) playAudioDirect(video VideoInfo) error {
	// Obtener información del video
	result, err := goutubedl.New(context.Background(), video.URL, goutubedl.Options{})
	if err != nil {
		return fmt.Errorf("error al obtener video: %v", err)
	}

	// Descargar audio
	downloadResult, err := result.Download(context.Background(), "bestaudio/best")
	if err != nil {
		return fmt.Errorf("error al descargar audio: %v", err)
	}

	// Usar ffplay directamente para reproducir el stream
	ffplayCmd := exec.Command("ffplay",
		"-nodisp",            // No mostrar ventana
		"-autoexit",          // Salir automáticamente al terminar
		"-loglevel", "quiet", // Silencioso
		"-i", "pipe:0", // Leer desde stdin
	)

	// Conectar stdin
	ffplayCmd.Stdin = downloadResult
	ffplayCmd.Stdout = os.Stdout
	ffplayCmd.Stderr = os.Stderr

	// Iniciar reproducción
	if err := ffplayCmd.Start(); err != nil {
		downloadResult.Close()
		return fmt.Errorf("error iniciando ffplay: %v", err)
	}

	mp.mu.Lock()
	mp.currentProcess = ffplayCmd.Process
	mp.mu.Unlock()

	// Esperar a que termine la reproducción
	err = ffplayCmd.Wait()
	downloadResult.Close()

	mp.mu.Lock()
	mp.currentProcess = nil
	mp.mu.Unlock()

	if err != nil && !strings.Contains(err.Error(), "exit status") {
		return fmt.Errorf("error en reproducción: %v", err)
	}

	return nil
}

func (mp *MusicPlayer) ShowQueue() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	fmt.Println("\n🎵 Cola de Reproducción:")
	if len(mp.playlist) == 0 {
		fmt.Println("   La cola está vacía")
		return
	}

	for i, video := range mp.playlist {
		status := "  "
		if i == mp.currentIndex {
			status = "▶️"
		}
		fmt.Printf("   %s %d. %s\n", status, i+1, video.Title)
		fmt.Printf("      👤 %s\n", video.Requester)
	}
	fmt.Printf("\n   Total: %d canciones en cola\n", len(mp.playlist))
}

func main() {
	fmt.Println("🎵 Bot de Música Simplificado")
	fmt.Println("==============================")
	fmt.Println("Comandos disponibles:")
	fmt.Println("!play [canción] - Añadir canción a la cola")
	fmt.Println("!revoke - Revocar tu última canción")
	fmt.Println("!skip - Saltar canción actual")
	fmt.Println("!queue - Mostrar cola actual")
	fmt.Println("!exit - Salir del programa")

	// Verificar que ffplay está disponible
	if err := exec.Command("ffplay", "-version").Run(); err != nil {
		log.Fatal("❌ ffplay no encontrado. Instala ffmpeg para continuar.")
	}

	player := NewMusicPlayer()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(input)

		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])

		switch command {
		case "!play":
			if len(parts) < 2 {
				fmt.Println("❌ Uso: !play [nombre de la canción]")
				continue
			}
			songName := strings.Join(parts[1:], " ")
			if err := player.PlaySong(songName, "Usuario"); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "!revoke":
			if err := player.RevokeSong("Usuario"); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "!skip":
			if err := player.Skip(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				fmt.Println("✅ Saltando canción actual...")
			}

		case "!queue":
			player.ShowQueue()

		case "!exit":
			fmt.Println("👋 ¡Hasta luego!")
			return

		default:
			fmt.Println("❌ Comando no reconocido. Comandos: !play, !revoke, !skip, !queue, !exit")
		}
	}
}
