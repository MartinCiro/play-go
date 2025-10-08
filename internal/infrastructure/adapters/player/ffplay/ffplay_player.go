package ffplay

import (
	"context"
	"fmt"
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

func (f *FFPlayPlayer) Play(stream goutubedl.Result) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Obtener información del video
	result, err := goutubedl.New(context.Background(), stream.Info.WebpageURL, goutubedl.Options{})
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
		"-nodisp",
		"-autoexit",
		"-loglevel", "quiet",
		"-i", "pipe:0",
	)

	ffplayCmd.Stdin = downloadResult
	ffplayCmd.Stdout = os.Stdout
	ffplayCmd.Stderr = os.Stderr

	if err := ffplayCmd.Start(); err != nil {
		downloadResult.Close()
		return fmt.Errorf("error iniciando ffplay: %v", err)
	}

	f.currentProcess = ffplayCmd.Process

	// Esperar en goroutine para no bloquear
	go func() {
		err := ffplayCmd.Wait()
		downloadResult.Close()

		f.mu.Lock()
		f.currentProcess = nil
		f.mu.Unlock()

		if err != nil && !strings.Contains(err.Error(), "exit status") {
			fmt.Printf("❌ Error en reproducción: %v\n", err)
		}
	}()

	return nil
}

func (f *FFPlayPlayer) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.currentProcess != nil {
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
