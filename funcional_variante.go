package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
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

	cmd := exec.Command("yt-dlp",
		"--skip-download",
		"--print-json",
		url)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error al obtener información del video: %v", err)
	}

	var videoInfo2 VideoInfo
	if err := json.Unmarshal(output, &videoInfo2); err != nil {
		return fmt.Errorf("error al parsear información del video: %v", err)
	}

	// Evitar duplicados en la playlist
	for _, v := range mp.playlist {
		if v.ID == videoInfo2.ID {
			fmt.Println("⚠️ La canción ya está en la playlist")
			return nil
		}
	}

	mp.playlist = append(mp.playlist, videoInfo2)
	fmt.Printf("✓ Añadido: %s\n", videoInfo2.Title)
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

		ffplayCmd, ytdlpCmd, err := mp.startPlayback(video)
		if err != nil {
			log.Printf("Error iniciando reproducción de %s: %v", video.Title, err)
			currentIndex++
			continue
		}

		fmt.Println("\nControles: [n] Siguiente canción, [s] Detener reproducción")

		stop := mp.waitForPlayback(ffplayCmd, ytdlpCmd)

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

func (mp *MusicPlayer) startPlayback(video VideoInfo) (*exec.Cmd, *exec.Cmd, error) {
	ytdlpCmd := exec.Command("yt-dlp",
		"-x",
		"--audio-format", "best",
		"-o", "-",
		"--quiet",
		video.URL)

	ffplayCmd := exec.Command("ffplay",
		"-nodisp",
		"-autoexit",
		"-loglevel", "quiet",
		"-i", "pipe:0")

	pipe, err := ytdlpCmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("error creando pipe: %v", err)
	}
	ffplayCmd.Stdin = pipe

	if err := ytdlpCmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("error iniciando yt-dlp: %v", err)
	}

	if err := ffplayCmd.Start(); err != nil {
		_ = ytdlpCmd.Process.Kill()
		return nil, nil, fmt.Errorf("error iniciando ffplay: %v", err)
	}

	return ffplayCmd, ytdlpCmd, nil
}

func (mp *MusicPlayer) kill(ffplayCmd *exec.Cmd, ytdlpCmd *exec.Cmd) {
	// Matar ffplay con taskkill
	if ffplayCmd != nil && ffplayCmd.Process != nil {
		exec.Command("taskkill", "/IM", "ffplay.exe", "/F").Run()
	}

	// Matar yt-dlp con taskkill
	if ytdlpCmd != nil && ytdlpCmd.Process != nil {
		exec.Command("taskkill", "/IM", "yt-dlp.exe", "/F").Run()
	}

	// Esperar un poco para que los procesos terminen
	time.Sleep(100 * time.Millisecond)
}

func (mp *MusicPlayer) waitForPlayback(ffplayCmd *exec.Cmd, ytdlpCmd *exec.Cmd) bool {
	done := make(chan struct{})
	userInput := make(chan string, 1)

	go func() {
		defer close(done)
		err := ffplayCmd.Wait()
		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			log.Printf("Error en reproducción: %v", err)
		}
		mp.kill(nil, ytdlpCmd)
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
				mp.kill(ffplayCmd, ytdlpCmd)
				return false
			case "s":
				fmt.Println("⏹️ Deteniendo reproducción...")
				mp.kill(ffplayCmd, ytdlpCmd)
				return true
			default:
				fmt.Println("Comando no reconocido. Usa: [n] Siguiente, [s] Detener")
			}
		}
	}
}

func (mp *MusicPlayer) ClearPlaylist2() {
	mp.playlist = nil // más idiomático que make([]VideoInfo,0)
	fmt.Println("🗑️ Playlist limpiada")
}

func main() {
	fmt.Println("🎵 YouTube Music Player en Go!")
	fmt.Println("===============================")
	fmt.Println("Requisitos: yt-dlp y ffmpeg deben estar instalados")

	if err := checkDependencies2(); err != nil {
		log.Fatalf("❌ Error de dependencias: %v", err)
	}

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
			player.ClearPlaylist2()
		case "5":
			fmt.Println("👋 ¡Hasta luego!")
			return
		default:
			fmt.Println("❌ Opción no válida")
		}
	}
}

func checkDependencies2() error {
	if err := exec.Command("yt-dlp", "--version").Run(); err != nil {
		return fmt.Errorf("yt-dlp no encontrado. Instala con: pip install yt-dlp")
	}
	if err := exec.Command("ffplay", "-version").Run(); err != nil {
		return fmt.Errorf("ffplay no encontrado. Instala ffmpeg")
	}
	fmt.Println("✅ Dependencias verificadas: yt-dlp y ffplay disponibles")
	return nil
}
