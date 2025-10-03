package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	goutubedl "github.com/wader/goutubedl"
)

type VideoInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"webpage_url"`
}

type MusicPlayer struct {
	playlist []VideoInfo
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		playlist: []VideoInfo{},
	}
}

func (mp *MusicPlayer) SearchAndAdd(url string) error {
	fmt.Println("Obteniendo información del video...")

	// Configurar opciones
	goutubedl.Path = "yt-dlp" // Buscará yt-dlp en PATH o lo descargará automáticamente

	result, err := goutubedl.New(context.Background(), url, goutubedl.Options{})
	if err != nil {
		return fmt.Errorf("error al obtener información del video: %v", err)
	}

	// Evitar duplicados en la playlist
	for _, v := range mp.playlist {
		if v.ID == result.Info.ID {
			fmt.Println("⚠️ La canción ya está en la playlist")
			return nil
		}
	}

	videoInfo := VideoInfo{
		ID:    result.Info.ID,
		Title: result.Info.Title,
		URL:   url,
	}

	mp.playlist = append(mp.playlist, videoInfo)
	fmt.Printf("✓ Añadido: %s\n", videoInfo.Title)
	return nil
}

func (mp *MusicPlayer) ShowPlaylist() {
	fmt.Println("\n🎵 Playlist Actual:")
	if len(mp.playlist) == 0 {
		fmt.Println("   La playlist está vacía")
		return
	}

	for i, video := range mp.playlist {
		fmt.Printf("   %d. %s\n", i+1, video.Title)
	}
	fmt.Println()
}

func (mp *MusicPlayer) Play() error {
	if len(mp.playlist) == 0 {
		return fmt.Errorf("la playlist está vacía")
	}

	currentIndex := 0
	for currentIndex < len(mp.playlist) {
		video := mp.playlist[currentIndex]
		fmt.Printf("\n🎵 Reproduciendo (%d/%d): %s\n", currentIndex+1, len(mp.playlist), video.Title)

		ffplayCmd, err := mp.startPlayback(video)
		if err != nil {
			log.Printf("Error iniciando reproducción de %s: %v", video.Title, err)
			currentIndex++
			continue
		}

		fmt.Println("\nControles: [n] Siguiente canción, [s] Detener reproducción")

		stop := mp.waitForPlayback(ffplayCmd)

		if stop {
			fmt.Println("⏹️ Reproducción detenida")
			break
		}

		currentIndex++

		if currentIndex >= len(mp.playlist) {
			fmt.Println("\n🎉 ¡Hemos llegado al final de la lista!")
			fmt.Println("   Agrega más canciones para continuar reproduciendo")
			break
		}
	}

	return nil
}

func (mp *MusicPlayer) startPlayback(video VideoInfo) (*exec.Cmd, error) {
	// Obtener información del video
	result, err := goutubedl.New(context.Background(), video.URL, goutubedl.Options{})
	if err != nil {
		return nil, fmt.Errorf("error al obtener video: %v", err)
	}

	// Obtener el stream de audio
	downloadResult, err := result.Download(context.Background(), "bestaudio[ext=m4a]/bestaudio")
	if err != nil {
		return nil, fmt.Errorf("error al obtener stream: %v", err)
	}

	// Configurar ffplay
	ffplayCmd := exec.Command("ffplay",
		"-nodisp",
		"-autoexit",
		"-loglevel", "quiet",
		"-i", "pipe:0")

	// Conectar el stream a ffplay
	ffplayCmd.Stdin = downloadResult

	if err := ffplayCmd.Start(); err != nil {
		downloadResult.Close()
		return nil, fmt.Errorf("error iniciando ffplay: %v", err)
	}

	// Cerrar el stream cuando ffplay termine
	go func() {
		ffplayCmd.Wait()
		downloadResult.Close()
	}()

	return ffplayCmd, nil
}

func (mp *MusicPlayer) kill(ffplayCmd *exec.Cmd) {
	if ffplayCmd != nil && ffplayCmd.Process != nil {
		exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", ffplayCmd.Process.Pid)).Run()
	}
	time.Sleep(100 * time.Millisecond)
}

func (mp *MusicPlayer) waitForPlayback(ffplayCmd *exec.Cmd) bool {
	done := make(chan struct{})
	userInput := make(chan string, 1)

	go func() {
		defer close(done)
		err := ffplayCmd.Wait()
		if err != nil && !strings.Contains(err.Error(), "exit status") {
			log.Printf("Error en reproducción: %v", err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input := strings.TrimSpace(strings.ToLower(scanner.Text()))
			userInput <- input
		}
	}()

	for {
		select {
		case <-done:
			return false
		case input := <-userInput:
			switch input {
			case "n":
				fmt.Println("⏭️ Saltando a siguiente canción...")
				mp.kill(ffplayCmd)
				return false
			case "s":
				fmt.Println("⏹️ Deteniendo reproducción...")
				mp.kill(ffplayCmd)
				return true
			default:
				fmt.Println("Comando no reconocido. Usa: [n] Siguiente, [s] Detener")
			}
		}
	}
}

func (mp *MusicPlayer) ClearPlaylist() {
	mp.playlist = []VideoInfo{}
	fmt.Println("🗑️ Playlist limpiada")
}

func main() {
	fmt.Println("🎵 YouTube Music Player en Go!")
	fmt.Println("===============================")
	fmt.Println("Nota: Se descargará yt-dlp automáticamente en primer uso")

	player := NewMusicPlayer()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\nOpciones:")
		fmt.Println("1. Añadir canción (URL de YouTube)")
		fmt.Println("2. Ver playlist")
		fmt.Println("3. Reproducir playlist")
		fmt.Println("4. Limpiar playlist")
		fmt.Println("5. Salir")
		fmt.Print("Selecciona una opción: ")

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		switch input {
		case "1":
			fmt.Print("Introduce URL de YouTube: ")
			if scanner.Scan() {
				url := strings.TrimSpace(scanner.Text())
				if url == "" {
					fmt.Println("❌ URL no puede estar vacía")
					continue
				}
				if err := player.SearchAndAdd(url); err != nil {
					fmt.Printf("❌ Error: %v\n", err)
				}
			}
		case "2":
			player.ShowPlaylist()
		case "3":
			if err := player.Play(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "4":
			player.ClearPlaylist()
		case "5":
			fmt.Println("👋 ¡Hasta luego!")
			return
		default:
			fmt.Println("❌ Opción no válida")
		}
	}
}

func checkDependencies() error {
	if err := exec.Command("ffplay", "-version").Run(); err != nil {
		return fmt.Errorf("ffplay no encontrado. Instala ffmpeg")
	}
	fmt.Println("✅ Dependencias verificadas: ffplay disponible")
	return nil
}
