package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/wader/goutubedl"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Uso: go run main.go \"nombre de la canción\"")
	}

	songName := strings.Join(os.Args[1:], " ")
	ctx := context.Background()

	// Usar ytsearch para obtener el primer resultado
	searchQuery := "ytsearch1:" + songName

	result, err := goutubedl.New(ctx, searchQuery, goutubedl.Options{})
	if err != nil {
		log.Fatalf("Error en la búsqueda: %v", err)
	}

	// Para ytsearch, necesitamos procesar el resultado como playlist
	// No es necesario comprobar si result.Info es nil porque es un struct
	// Si tiene entries, es una playlist de resultados
	if len(result.Info.Entries) > 0 {
		// Obtener la URL del primer resultado
		firstResult := result.Info.Entries[0]
		if firstResult.WebpageURL != "" {
			fmt.Println(firstResult.WebpageURL)
			return
		}
	}

	// Si no tiene entries pero tiene WebpageURL directa
	if result.Info.WebpageURL != "" {
		fmt.Println(result.Info.WebpageURL)
		return
	}

	// Si llegamos aquí, intentemos obtener la URL de forma diferente
	if result.Info.ID != "" {
		// Construir la URL manualmente
		url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", result.Info.ID)
		fmt.Println(url)
		return
	}

	log.Fatal("No se pudo obtener la URL del video")
}
