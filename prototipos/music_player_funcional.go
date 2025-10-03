package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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
		playlist: make([]VideoInfo, 0),
	}
}

func (mp *MusicPlayer) SearchAndAdd(url string) error {
	fmt.Printf("Obteniendo información del video...\n")

	// Usar yt-dlp para obtener información del video
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
	fmt.Print(videoInfo)
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

	for i, video := range mp.playlist {
		fmt.Printf("\n🎵 Reproduciendo (%d/%d): %s\n", i+1, len(mp.playlist), video.Title)

		err := mp.playVideo(video)
		if err != nil {
			log.Printf("Error reproduciendo %s: %v", video.Title, err)

			// Preguntar si continuar
			fmt.Print("¿Continuar con la siguiente canción? (s/n): ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				if strings.ToLower(scanner.Text()) != "s" {
					break
				}
			}
			continue
		}
	}

	fmt.Println("\n✅ Playlist completada")
	return nil
}

func (mp *MusicPlayer) playVideo(video VideoInfo) error {
	// Método 1: Intentar con yt-dlp + ffplay
	fmt.Println("🔊 Cargando audio...")

	cmd := exec.Command("yt-dlp",
		"-x",                     // Extraer audio
		"--audio-format", "best", // Mejor formato de audio
		"-o", "-", // Output a stdout
		"--quiet", // Menos output
		video.URL)

	ffplayCmd := exec.Command("ffplay",
		"-nodisp",            // No mostrar ventana
		"-autoexit",          // Salir automáticamente
		"-loglevel", "quiet", // Silenciar logs
		"-i", "pipe:0") // Leer de stdin

	// Conectar los pipes
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creando pipe: %v", err)
	}
	ffplayCmd.Stdin = pipe

	// Iniciar los comandos
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error iniciando yt-dlp: %v", err)
	}
	if err := ffplayCmd.Start(); err != nil {
		// Si ffplay no funciona, intentar método alternativo
		return mp.alternativePlay(video)
	}

	// Esperar a que terminen
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error en yt-dlp: %v", err)
	}
	if err := ffplayCmd.Wait(); err != nil {
		return fmt.Errorf("error en ffplay: %v", err)
	}

	return nil
}

func (mp *MusicPlayer) alternativePlay(video VideoInfo) error {
	fmt.Println("🎵 Usando método alternativo...")

	// Método alternativo: Descargar temporalmente y reproducir
	cmd := exec.Command("yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"--exec", "ffplay -nodisp -autoexit -loglevel quiet {}",
		"--no-keep-video",
		video.URL)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error en método alternativo: %v", err)
	}

	return nil
}

func (mp *MusicPlayer) ClearPlaylist() {
	mp.playlist = make([]VideoInfo, 0)
	fmt.Println("🗑️ Playlist limpiada")
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
