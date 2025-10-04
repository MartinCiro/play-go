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
	playlist       []VideoInfo
	currentProcess *os.Process
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		playlist: []VideoInfo{},
	}
}

func (mp *MusicPlayer) SearchAndAdd(url string) error {
	fmt.Println("Obteniendo información del video...")

	result, err := goutubedl.New(context.Background(), url, goutubedl.Options{})
	if err != nil {
		return fmt.Errorf("error al obtener información del video: %v", err)
	}

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

func (mp *MusicPlayer) Play() error {
	if len(mp.playlist) == 0 {
		return fmt.Errorf("la playlist está vacía")
	}

	for i, video := range mp.playlist {
		fmt.Printf("\n🎵 Reproduciendo (%d/%d): %s\n", i+1, len(mp.playlist), video.Title)

		if err := mp.playAudioDirect(video); err != nil {
			fmt.Printf("❌ Error reproduciendo %s: %v\n", video.Title, err)
			continue
		}

		fmt.Println("✓ Canción completada")
	}

	fmt.Println("\n🎉 ¡Hemos llegado al final de la lista!")
	return nil
}

func (mp *MusicPlayer) playAudioDirect(video VideoInfo) error {
	// Obtener información del video
	result, err := goutubedl.New(context.Background(), video.URL, goutubedl.Options{})
	if err != nil {
		return fmt.Errorf("error al obtener video: %v", err)
	}

	fmt.Println("📥 Descargando y reproduciendo audio...")

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

	mp.currentProcess = ffplayCmd.Process

	// Canal para control
	done := make(chan error, 1)

	// Esperar a que termine ffplay
	go func() {
		err := ffplayCmd.Wait()
		downloadResult.Close()
		done <- err
	}()

	fmt.Println("Controles: [n] Siguiente, [s] Detener")

	// Manejar controles
	return mp.handlePlaybackControls(done)
}

func (mp *MusicPlayer) handlePlaybackControls(done chan error) error {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {
		case err := <-done:
			// Reproducción terminó naturalmente
			if err != nil && !strings.Contains(err.Error(), "exit status") {
				return fmt.Errorf("error en reproducción: %v", err)
			}
			return nil
		default:
			if scanner.Scan() {
				cmd := strings.TrimSpace(strings.ToLower(scanner.Text()))
				switch cmd {
				case "n":
					fmt.Println("⏭️ Saltando a siguiente canción...")
					mp.Stop()
					return nil
				case "s":
					fmt.Println("⏹️ Deteniendo reproducción...")
					mp.Stop()
					return fmt.Errorf("reproducción detenida por el usuario")
				default:
					fmt.Println("Comando no reconocido. Usa: [n] Siguiente, [s] Detener")
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (mp *MusicPlayer) Stop() {
	if mp.currentProcess != nil {
		// En Windows
		exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", mp.currentProcess.Pid)).Run()
		// En Linux/Mac: mp.currentProcess.Signal(os.Interrupt)
		mp.currentProcess = nil
	}
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

func (mp *MusicPlayer) ClearPlaylist() {
	mp.Stop()
	mp.playlist = []VideoInfo{}
	fmt.Println("🗑️ Playlist limpiada")
}

func main() {
	fmt.Println("🎵 YouTube Music Player en Go!")
	fmt.Println("===============================")
	fmt.Println("Nota: Usando ffplay para reproducción directa")

	// Verificar que ffplay está disponible
	if err := exec.Command("ffplay", "-version").Run(); err != nil {
		log.Fatal("❌ ffplay no encontrado. Instala ffmpeg para continuar.")
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
				if url != "" {
					if err := player.SearchAndAdd(url); err != nil {
						fmt.Printf("❌ Error: %v\n", err)
					}
				}
			}
		case "2":
			player.ShowPlaylist()
		case "3":
			if err := player.Play(); err != nil {
				if err.Error() != "reproducción detenida por el usuario" {
					fmt.Printf("❌ Error: %v\n", err)
				}
			}
		case "4":
			player.ClearPlaylist()
		case "5":
			player.Stop()
			fmt.Println("👋 ¡Hasta luego!")
			return
		default:
			fmt.Println("❌ Opción no válida")
		}
	}
}
