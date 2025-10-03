package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type MusicPlayer struct {
	playlist []string
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		playlist: make([]string, 0),
	}
}

func (mp *MusicPlayer) AddToPlaylist(url string) {
	mp.playlist = append(mp.playlist, url)
	fmt.Printf("✓ Añadido: %s\n", url)
}

func (mp *MusicPlayer) ShowPlaylist() {
	fmt.Println("\n🎵 Playlist Actual:")
	if len(mp.playlist) == 0 {
		fmt.Println("   La playlist está vacía")
		return
	}

	for i, url := range mp.playlist {
		fmt.Printf("   %d. %s\n", i+1, url)
	}
	fmt.Println()
}

func (mp *MusicPlayer) Play() error {
	if len(mp.playlist) == 0 {
		return fmt.Errorf("la playlist está vacía")
	}

	// Crear directorio temporal para descargas
	tempDir := filepath.Join(".", "temp_music")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	for i, url := range mp.playlist {
		fmt.Printf("\n🎵 Reproduciendo (%d/%d)\n", i+1, len(mp.playlist))

		// Descargar audio
		outputFile := filepath.Join(tempDir, fmt.Sprintf("audio_%d.mp3", i))
		cmd := exec.Command("yt-dlp",
			"-x",
			"--audio-format", "mp3",
			"-o", outputFile,
			url)

		fmt.Println("📥 Descargando audio...")
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Error descargando: %v\n", err)
			continue
		}

		// Reproducir con reproductor del sistema
		fmt.Println("🔊 Reproduciendo...")
		if err := mp.playWithSystemPlayer(outputFile); err != nil {
			fmt.Printf("❌ Error reproduciendo: %v\n", err)
			continue
		}

		// Limpiar archivo después de reproducir
		os.Remove(outputFile)
	}

	fmt.Println("\n✅ Playlist completada")
	return nil
}

func (mp *MusicPlayer) playWithSystemPlayer(file string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "/wait", file)
	case "darwin":
		cmd = exec.Command("afplay", file)
	case "linux":
		// Intentar varios reproductores comunes en Linux
		for _, player := range []string{"mpg123", "ffplay", "play"} {
			if _, err := exec.LookPath(player); err == nil {
				if player == "ffplay" {
					cmd = exec.Command(player, "-nodisp", "-autoexit", file)
				} else {
					cmd = exec.Command(player, file)
				}
				break
			}
		}
		if cmd == nil {
			return fmt.Errorf("no se encontró reproductor de audio")
		}
	default:
		return fmt.Errorf("sistema operativo no soportado")
	}

	return cmd.Run()
}

func (mp *MusicPlayer) ClearPlaylist() {
	mp.playlist = make([]string, 0)
	fmt.Println("🗑️ Playlist limpiada")
}

func main() {
	fmt.Println("🎵 YouTube Music Player Semi-Portable")
	fmt.Println("======================================")
	fmt.Println("Nota: Requiere yt-dlp pero usa reproductores del sistema")

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
				player.AddToPlaylist(url)
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
