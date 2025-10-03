package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type VideoInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"webpage_url"`
}

type MusicPlayer struct {
	playlist    []VideoInfo
	currentCmd  *exec.Cmd
	isPlaying   bool
	isPaused    bool
	currentSong int
	stopChan    chan bool
	pauseChan   chan bool
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		playlist:    make([]VideoInfo, 0),
		stopChan:    make(chan bool, 1),
		pauseChan:   make(chan bool, 1),
		currentSong: -1,
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
		if i == mp.currentSong {
			if mp.isPlaying {
				if mp.isPaused {
					status = "⏸️"
				} else {
					status = "▶️"
				}
			} else {
				status = "⏹️"
			}
		}
		fmt.Printf("   %d. %s %s\n", i+1, status, video.Title)
	}
	fmt.Println()
}

func (mp *MusicPlayer) Play() error {
	if len(mp.playlist) == 0 {
		return fmt.Errorf("la playlist está vacía")
	}

	// Si ya hay una canción reproduciéndose, continuar desde ahí
	if mp.currentSong == -1 {
		mp.currentSong = 0
	}

	// Configurar manejo de señales para limpiar procesos
	mp.setupSignalHandler()

	fmt.Println("\n🎵 Controles:")
	fmt.Println("   [p] Pausar/Reanudar")
	fmt.Println("   [s] Parar")
	fmt.Println("   [n] Siguiente canción")
	fmt.Println("   [b] Canción anterior")
	fmt.Println("   [q] Volver al menú principal")

	for mp.currentSong < len(mp.playlist) {
		if mp.stopChan == nil {
			mp.stopChan = make(chan bool, 1)
		}

		video := mp.playlist[mp.currentSong]
		fmt.Printf("\n🎵 Reproduciendo (%d/%d): %s\n", mp.currentSong+1, len(mp.playlist), video.Title)

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
		}

		// Verificar si fue detenido por el usuario
		select {
		case <-mp.stopChan:
			fmt.Println("\n⏹️ Reproducción detenida")
			return nil
		default:
			// Continuar con la siguiente canción
		}

		mp.currentSong++
	}

	fmt.Println("\n✅ Playlist completada")
	mp.currentSong = -1
	mp.isPlaying = false
	mp.isPaused = false
	return nil
}

func (mp *MusicPlayer) playVideo(video VideoInfo) error {
	mp.isPlaying = true
	mp.isPaused = false

	// Usar un enfoque más simple: descargar y reproducir directamente
	fmt.Println("🔊 Cargando audio...")

	// Crear archivo temporal
	tmpFile := fmt.Sprintf("temp_audio_%d.mp3", time.Now().Unix())
	defer os.Remove(tmpFile) // Limpiar al terminar

	// Primero descargar el audio
	downloadCmd := exec.Command("yt-dlp",
		"-x",                    // Extraer audio
		"--audio-format", "mp3", // Formato mp3
		"-o", tmpFile, // Output a archivo
		"--quiet", // Menos output
		video.URL)

	fmt.Print("Descargando... ")
	if err := downloadCmd.Run(); err != nil {
		return fmt.Errorf("error descargando audio: %v", err)
	}
	fmt.Println("✓")

	// Ahora reproducir con ffplay
	ffplayCmd := exec.Command("ffplay",
		"-nodisp",            // No mostrar ventana
		"-autoexit",          // Salir automáticamente al terminar
		"-loglevel", "quiet", // Silenciar logs
		tmpFile)

	if err := ffplayCmd.Start(); err != nil {
		return fmt.Errorf("error iniciando ffplay: %v", err)
	}

	mp.currentCmd = ffplayCmd

	// Goroutine para manejar entrada del usuario durante la reproducción
	go mp.handlePlaybackControls(ffplayCmd)

	// Esperar a que termine la reproducción o sea interrumpida
	err := ffplayCmd.Wait()

	mp.currentCmd = nil
	mp.isPlaying = false
	mp.isPaused = false

	if err != nil {
		// ffplay puede devolver error cuando se interrumpe, eso es normal
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				return fmt.Errorf("error en reproducción: %v", err)
			}
		}
	}

	return nil
}

func (mp *MusicPlayer) handlePlaybackControls(cmd *exec.Cmd) {
	scanner := bufio.NewScanner(os.Stdin)

	for mp.isPlaying && cmd.Process != nil {
		if scanner.Scan() {
			input := strings.TrimSpace(strings.ToLower(scanner.Text()))

			switch input {
			case "p": // Pausar/Reanudar
				mp.TogglePause()
			case "s": // Parar
				mp.Stop()
				return
			case "n": // Siguiente
				mp.Next()
				return
			case "b": // Anterior
				mp.Previous()
				return
			case "q": // Salir al menú
				mp.Stop()
				return
			default:
				fmt.Println("Comando no reconocido. Usa: p (pausa), s (stop), n (siguiente), b (anterior), q (salir)")
			}
		}

		// Pequeña pausa para no saturar la CPU
		time.Sleep(100 * time.Millisecond)

		// Verificar si el proceso aún está corriendo
		if cmd.Process != nil {
			if _, err := os.FindProcess(cmd.Process.Pid); err != nil {
				break
			}
		}
	}
}

func (mp *MusicPlayer) TogglePause() {
	if mp.currentCmd == nil || mp.currentCmd.Process == nil {
		return
	}

	// En Windows no podemos pausar fácilmente un proceso externo
	// Implementamos un método alternativo: mostrar mensaje y simular pausa
	if mp.isPaused {
		mp.isPaused = false
		fmt.Println("▶️ Reanudado")
	} else {
		mp.isPaused = true
		fmt.Println("⏸️ Pausado (Nota: En Windows la pausa es simulada)")
		fmt.Println("   Para continuar presiona 'p' nuevamente")
	}
}

func (mp *MusicPlayer) Stop() {
	mp.isPlaying = false
	mp.isPaused = false
	if mp.currentCmd != nil && mp.currentCmd.Process != nil {
		// Matar el proceso de ffplay
		if runtime.GOOS == "windows" {
			mp.currentCmd.Process.Kill()
		} else {
			mp.currentCmd.Process.Signal(syscall.SIGTERM)
		}
	}
	if mp.stopChan != nil {
		select {
		case mp.stopChan <- true:
		default:
		}
	}
}

func (mp *MusicPlayer) Next() {
	if mp.currentSong < len(mp.playlist)-1 {
		fmt.Println("⏭️ Siguiente canción...")
		mp.Stop()
		// La siguiente canción se reproducirá automáticamente en el bucle principal
	} else {
		fmt.Println("❌ No hay más canciones en la playlist")
	}
}

func (mp *MusicPlayer) Previous() {
	if mp.currentSong > 0 {
		fmt.Println("⏮️ Canción anterior...")
		mp.Stop()
		mp.currentSong-- // Retroceder para que el bucle principal reproduzca la anterior
		if mp.currentSong < 0 {
			mp.currentSong = 0
		}
	} else {
		fmt.Println("❌ Esta es la primera canción")
	}
}

func (mp *MusicPlayer) setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n🛑 Interrupción recibida, limpiando...")
		mp.Stop()
		os.Exit(1)
	}()
}

func (mp *MusicPlayer) ClearPlaylist() {
	mp.Stop()
	mp.playlist = make([]VideoInfo, 0)
	mp.currentSong = -1
	fmt.Println("🗑️ Playlist limpiada")
}

func (mp *MusicPlayer) GetStatus() string {
	if !mp.isPlaying {
		return "Detenido"
	}
	if mp.isPaused {
		if mp.currentSong >= 0 && mp.currentSong < len(mp.playlist) {
			return fmt.Sprintf("Pausado: %s", mp.playlist[mp.currentSong].Title)
		}
		return "Pausado"
	}
	if mp.currentSong >= 0 && mp.currentSong < len(mp.playlist) {
		return fmt.Sprintf("Reproduciendo: %s", mp.playlist[mp.currentSong].Title)
	}
	return "Reproduciendo"
}

func main() {
	fmt.Println("🎵 YouTube Music Player en Go!")
	fmt.Println("===============================")
	fmt.Println("Requisitos: yt-dlp y ffmpeg deben estar instalados")

	// Verificar dependencias
	if err := checkMusicPlayerDependencies(); err != nil {
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
		fmt.Println("5. Estado actual")
		fmt.Println("6. Salir")
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
			fmt.Printf("📊 Estado: %s\n", player.GetStatus())

		case "6":
			player.Stop()
			fmt.Println("👋 ¡Hasta luego!")
			return

		default:
			fmt.Println("❌ Opción no válida")
		}
	}
}

func checkMusicPlayerDependencies() error {
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
