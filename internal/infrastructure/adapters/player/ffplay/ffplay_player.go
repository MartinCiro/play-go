package ffplay

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	goutubedl "github.com/wader/goutubedl"
)

type FFPlayPlayer struct {
	currentProcess *os.Process
	mu             sync.Mutex
}

func NewFFPlayPlayer() *FFPlayPlayer {
	return &FFPlayPlayer{}
}

func (f *FFPlayPlayer) Play(stream interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Convertir el stream al tipo correcto (manteniendo compatibilidad con tu código)
	var downloadResult io.ReadCloser
	switch s := stream.(type) {
	case goutubedl.Result:
		result, err := s.Download(context.Background(), "bestaudio/best")
		if err != nil {
			return fmt.Errorf("error al descargar audio: %v", err)
		}
		downloadResult = result
	case io.ReadCloser:
		downloadResult = s
	default:
		return fmt.Errorf("tipo de stream no soportado: %T", stream)
	}
	defer downloadResult.Close()

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
		return fmt.Errorf("error iniciando ffplay: %v", err)
	}

	f.currentProcess = ffplayCmd.Process

	// Esperar a que termine la reproducción
	err := ffplayCmd.Wait()

	f.currentProcess = nil

	if err != nil && !strings.Contains(err.Error(), "exit status") {
		return fmt.Errorf("error en reproducción: %v", err)
	}

	return nil
}

func (f *FFPlayPlayer) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.currentProcess != nil {
		// Detener el proceso actual
		cmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", f.currentProcess.Pid))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error deteniendo reproducción: %v", err)
		}
		f.currentProcess = nil
	}
	return nil
}

func (f *FFPlayPlayer) IsPlaying() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.currentProcess != nil
}
