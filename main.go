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
	playlist     []VideoInfo
	currentIndex int
	isPlaying    bool
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		playlist:     make([]VideoInfo, 0),
		currentIndex: -1,
	}
}

func (mp *MusicPlayer) SearchAndAdd(url string) error {
	fmt.Printf("Obteniendo información del video...\n")

	cmd := exec.Command("yt-dlp",
		"--skip-download",
		"--print-json",
		url)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error al obtener información del video: %v", err)
	}

	var videoInfo VideoInfo
	if err := json.Unmarshal(output, &videoInfo); err != nil {
		return fmt.Errorf("error al parsear información del video: %v", err)
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
		status := " "
		if i == mp.currentIndex {
			status = "▶"
		}
		fmt.Printf("   %s %d. %s\n", status, i+1, video.Title)
	}
	fmt.Println()
}

func (mp *MusicPlayer) Play() error {
	if len(mp.playlist) == 0 {
		return fmt.Errorf("la playlist está vacía")
	}

	// Si no hay canción actual, empezar desde la primera
	if mp.currentIndex == -1 {
		mp.currentIndex = 0
	}

	return mp.playCurrentSong()
}

func (mp *MusicPlayer) playCurrentSong() error {
	if mp.currentIndex < 0 || mp.currentIndex >= len(mp.playlist) {
		return fmt.Errorf("índice de canción inválido")
	}

	video := mp.playlist[mp.currentIndex]
	fmt.Printf("\n🎵 Reproduciendo (%d/%d): %s\n", mp.currentIndex+1, len(mp.playlist), video.Title)

	// Iniciar la reproducción con yt-dlp y ffplay
	cmd, ytdlpCmd := mp.startPlayback(video)
	if cmd == nil {
		return fmt.Errorf("error iniciando reproducción")
	}

	mp.isPlaying = true

	// Mostrar controles
	fmt.Println("\nControles: [p] Pausa, [n] Siguiente, [b] Anterior, [s] Detener, [q] Volver al menú")

	// Esperar a que termine la reproducción o reciba comando
	stop := mp.waitForPlayback(cmd, ytdlpCmd)

	mp.isPlaying = false

	if stop {
		fmt.Println("⏹️ Reproducción detenida")
		return nil
	}

	return nil
}

func (mp *MusicPlayer) startPlayback(video VideoInfo) (*exec.Cmd, *exec.Cmd) {
	// Comando yt-dlp para extraer audio
	ytdlpCmd := exec.Command("yt-dlp",
		"-x",                     // Extraer audio
		"--audio-format", "best", // Mejor formato de audio
		"-o", "-", // Output a stdout
		"--quiet", // Menos output
		video.URL)

	// Comando ffplay para reproducir
	ffplayCmd := exec.Command("ffplay",
		"-nodisp",            // No mostrar ventana
		"-autoexit",          // Salir automáticamente
		"-loglevel", "quiet", // Silenciar logs
		"-i", "pipe:0") // Leer de stdin

	// Conectar los pipes
	pipe, err := ytdlpCmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creando pipe: %v\n", err)
		return nil, nil
	}
	ffplayCmd.Stdin = pipe

	// Iniciar yt-dlp
	if err := ytdlpCmd.Start(); err != nil {
		fmt.Printf("Error iniciando yt-dlp: %v\n", err)
		return nil, nil
	}

	// Iniciar ffplay
	if err := ffplayCmd.Start(); err != nil {
		fmt.Printf("Error iniciando ffplay: %v\n", err)
		ytdlpCmd.Process.Kill()
		return nil, nil
	}

	return ffplayCmd, ytdlpCmd
}

func (mp *MusicPlayer) killProcesses(ffplayCmd *exec.Cmd, ytdlpCmd *exec.Cmd) {
	if ffplayCmd != nil && ffplayCmd.Process != nil {
		ffplayCmd.Process.Kill()
	}

	if ytdlpCmd != nil && ytdlpCmd.Process != nil {
		ytdlpCmd.Process.Kill()
	}

	time.Sleep(100 * time.Millisecond)
}

func (mp *MusicPlayer) waitForPlayback(ffplayCmd *exec.Cmd, ytdlpCmd *exec.Cmd) bool {
	done := make(chan bool, 1)
	userInput := make(chan string, 1)

	// Goroutine para esperar que termine la reproducción
	go func() {
		err := ffplayCmd.Wait()
		if err != nil {
			// Ignorar error de exit status 1 que es normal en ffplay
			if !strings.Contains(err.Error(), "exit status 1") {
				fmt.Printf("Error en reproducción: %v\n", err)
			}
		}
		mp.killProcesses(nil, ytdlpCmd)
		done <- true
	}()

	// Goroutine para leer entrada del usuario
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := strings.TrimSpace(strings.ToLower(scanner.Text()))
			userInput <- input
			break
		}
	}()

	// Esperar eventos
	for {
		select {
		case <-done:
			return false // Canción terminó naturalmente
		case input := <-userInput:
			switch input {
			case "p":
				fmt.Println("⏸️ Pausa no disponible en esta versión - usa [s] para detener")
			case "n":
				fmt.Println("⏭️ Saltando a siguiente canción...")
				mp.killProcesses(ffplayCmd, ytdlpCmd)
				mp.currentIndex++
				if mp.currentIndex >= len(mp.playlist) {
					fmt.Println("🎉 Fin de la playlist")
					mp.currentIndex = -1
					return true
				}
				return false
			case "b":
				fmt.Println("⏮️ Volviendo a canción anterior...")
				mp.killProcesses(ffplayCmd, ytdlpCmd)
				if mp.currentIndex > 0 {
					mp.currentIndex--
				}
				return false
			case "s":
				fmt.Println("⏹️ Deteniendo reproducción...")
				mp.killProcesses(ffplayCmd, ytdlpCmd)
				mp.currentIndex = -1
				return true
			case "q":
				fmt.Println("↩ Volviendo al menú principal...")
				mp.killProcesses(ffplayCmd, ytdlpCmd)
				return true
			default:
				fmt.Println("Comando no reconocido. Usa: [n] Siguiente, [b] Anterior, [s] Detener, [q] Volver")
			}
		}
	}
}

func (mp *MusicPlayer) Next() error {
	if mp.currentIndex+1 >= len(mp.playlist) {
		return fmt.Errorf("no hay más canciones en la lista")
	}
	mp.currentIndex++
	return nil
}

func (mp *MusicPlayer) Previous() error {
	if mp.currentIndex-1 < 0 {
		return fmt.Errorf("no hay canción anterior")
	}
	mp.currentIndex--
	return nil
}

func (mp *MusicPlayer) ClearPlaylist() {
	mp.playlist = make([]VideoInfo, 0)
	mp.currentIndex = -1
	fmt.Println("🗑️ Playlist limpiada")
}

func (mp *MusicPlayer) NowPlaying() {
	if mp.currentIndex < 0 || mp.currentIndex >= len(mp.playlist) {
		fmt.Println("No se está reproduciendo nada")
		return
	}

	current := mp.playlist[mp.currentIndex]
	fmt.Printf("\n🎵 Actual: %s\n", current.Title)
	fmt.Printf("   Posición: %d/%d\n", mp.currentIndex+1, len(mp.playlist))
}

func main() {
	fmt.Println("🎵 YouTube Music Player en Go!")
	fmt.Println("===============================")
	fmt.Println("Requisitos: yt-dlp y ffmpeg deben estar instalados")

	// Verificar dependencias
	if err := checkDependencies(); err != nil {
		log.Fatalf("❌ Error de dependencias: %v", err)
	}

	player := NewMusicPlayer()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\nOpciones:")
		fmt.Println("1. Añadir canción (URL de YouTube)")
		fmt.Println("2. Ver playlist")
		fmt.Println("3. Reproducir playlist")
		fmt.Println("4. Siguiente canción")
		fmt.Println("5. Canción anterior")
		fmt.Println("6. Información de canción actual")
		fmt.Println("7. Limpiar playlist")
		fmt.Println("8. Salir")
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
			if err := player.Next(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				fmt.Println("⏭️ Siguiente canción")
			}

		case "5":
			if err := player.Previous(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				fmt.Println("⏮️ Canción anterior")
			}

		case "6":
			player.NowPlaying()

		case "7":
			player.ClearPlaylist()

		case "8":
			fmt.Println("👋 ¡Hasta luego!")
			return

		default:
			fmt.Println("❌ Opción no válida")
		}
	}
}

func checkDependencies() error {
	// Verificar yt-dlp
	if err := exec.Command("yt-dlp", "--version").Run(); err != nil {
		return fmt.Errorf("yt-dlp no encontrado. Instala con: pip install yt-dlp")
	}

	// Verificar ffplay
	if err := exec.Command("ffplay", "-version").Run(); err != nil {
		return fmt.Errorf("ffplay no encontrado. Instala ffmpeg")
	}

	fmt.Println("✅ Dependencias verificadas: yt-dlp y ffplay disponibles")
	return nil
}
