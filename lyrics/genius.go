package lyrics

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type SyncedLyric struct {
	At    time.Duration
	Lyric string
	Index int
}

type Song struct {
	Description  string
	PlainLyrics  string
	SyncedLyrics []SyncedLyric
	LyricSource  string
	URL          string
}

var (
	ErrNotFound      = errors.New("canción no encontrada")
	ErrNetwork       = errors.New("error de conexión")
	ErrInvalidData   = errors.New("datos inválidos en la respuesta")
	ErrParsingFailed = errors.New("error al analizar el contenido")
)

// GetSongGenius obtiene las letras de una canción desde Genius.com
func GetSongGenius(artistName, trackName string) (Song, error) {
	s := Song{
		LyricSource: "Genius",
	}

	// Validar parámetros de entrada
	if strings.TrimSpace(artistName) == "" || strings.TrimSpace(trackName) == "" {
		return s, fmt.Errorf("%w: artista y título son requeridos", ErrInvalidData)
	}

	// Construir URL de manera más robusta
	url, err := buildGeniusURL(artistName, trackName)
	if err != nil {
		return s, fmt.Errorf("%w: %v", ErrInvalidData, err)
	}
	s.URL = url

	// Realizar petición HTTP con timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Get(url)
	if err != nil {
		return s, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return s, ErrNotFound
	}

	if res.StatusCode != http.StatusOK {
		return s, fmt.Errorf("error HTTP %d: %s", res.StatusCode, res.Status)
	}

	// Leer y parsear el contenido
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return s, fmt.Errorf("%w: error leyendo respuesta: %v", ErrNetwork, err)
	}

	content := string(b)

	// Extraer descripción usando múltiples métodos
	if desc, err := extractDescription(content); err == nil {
		s.Description = desc
	}

	// Extraer letras usando parser HTML robusto
	lyrics, err := extractLyrics(content)
	if err != nil {
		return s, fmt.Errorf("%w: %v", ErrParsingFailed, err)
	}

	if lyrics == "" {
		return s, ErrParsingFailed
	}

	s.PlainLyrics = lyrics
	return s, nil
}

// buildGeniusURL construye la URL de Genius de manera segura
func buildGeniusURL(artist, title string) (string, error) {
	// Limpiar y normalizar nombres
	cleanString := func(s string) string {
		// Remover caracteres problemáticos
		reg := regexp.MustCompile(`[!@#$%^&*()+=|;:'",<>?{}[\]\\/]`)
		s = reg.ReplaceAllString(s, "")

		// Reemplazar espacios y normalizar
		s = strings.TrimSpace(s)
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "-")
		s = strings.ReplaceAll(s, "&", "and")

		// Remover múltiples guiones
		reg = regexp.MustCompile(`-+`)
		s = reg.ReplaceAllString(s, "-")

		return strings.Trim(s, "-")
	}

	cleanArtist := cleanString(artist)
	cleanTitle := cleanString(title)

	if cleanArtist == "" || cleanTitle == "" {
		return "", ErrInvalidData
	}

	// Construir URL segura
	path := cleanArtist + "-" + cleanTitle + "-lyrics"
	return "https://genius.com/" + url.PathEscape(path), nil
}

// extractDescription extrae la descripción usando múltiples métodos
func extractDescription(content string) (string, error) {
	// Método 1: Meta tag og:description
	ogDescPattern := `<meta\s+property="og:description"\s+content="([^"]*)"`
	re := regexp.MustCompile(ogDescPattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return html.UnescapeString(matches[1]), nil
	}

	// Método 2: Meta tag con name="description"
	descPattern := `<meta\s+name="description"\s+content="([^"]*)"`
	re = regexp.MustCompile(descPattern)
	matches = re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return html.UnescapeString(matches[1]), nil
	}

	return "", ErrParsingFailed
}

// extractLyrics extrae las letras usando parser HTML robusto
func extractLyrics(content string) (string, error) {
	// Método 1: Buscar en el JavaScript preloaded state
	if lyrics, err := extractFromPreloadedState(content); err == nil && lyrics != "" {
		return lyrics, nil
	}

	// Método 2: Parsear HTML directamente
	if lyrics, err := parseHTMLContent(content); err == nil && lyrics != "" {
		return lyrics, nil
	}

	return "", ErrParsingFailed
}

// extractFromPreloadedState extrae letras del estado preloaded (método original mejorado)
func extractFromPreloadedState(content string) (string, error) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "window.__PRELOADED_STATE__") {
			// Extraer y limpiar el JSON
			start := strings.Index(line, "{")
			end := strings.LastIndex(line, "}") + 1
			if start == -1 || end == -1 {
				continue
			}

			jsonStr := line[start:end]
			jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")
			jsonStr = strings.ReplaceAll(jsonStr, "\\\\", "\\")

			// Buscar letras en el JSON
			return extractLyricsFromJSON(jsonStr), nil
		}
	}
	return "", ErrParsingFailed
}

// extractLyricsFromJSON extrae letras de diferentes estructuras JSON posibles
func extractLyricsFromJSON(jsonStr string) string {
	var lyrics strings.Builder

	// Patrón para buscar texto de letras
	patterns := []string{
		`"text":"([^"]+)"`,
		`"lyrics":"([^"]+)"`,
		`"body":"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jsonStr, -1)

		for _, match := range matches {
			if len(match) > 1 {
				text := match[1]
				// Filtrar texto que no son letras
				if isValidLyric(text) {
					lyrics.WriteString(html.UnescapeString(text))
					lyrics.WriteString("\n")
				}
			}
		}
	}

	return strings.TrimSpace(lyrics.String())
}

// parseHTMLContent parsea el HTML directamente para encontrar letras
func parseHTMLContent(content string) (string, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return "", err
	}

	var lyrics strings.Builder
	var foundLyrics bool

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			// Buscar divs que contengan letras
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "lyrics") {
					foundLyrics = true
					extractTextFromNode(n, &lyrics)
					return
				}
			}
		}

		// Buscar en elementos que comúnmente contienen letras
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "span") {
			for _, attr := range n.Attr {
				if attr.Key == "class" && (strings.Contains(attr.Val, "Lyrics__Container") ||
					strings.Contains(attr.Val, "lyrics")) {
					extractTextFromNode(n, &lyrics)
					foundLyrics = true
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	if foundLyrics {
		return cleanLyrics(lyrics.String()), nil
	}

	return "", ErrParsingFailed
}

// extractTextFromNode extrae texto de un nodo HTML
func extractTextFromNode(n *html.Node, builder *strings.Builder) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" && isValidLyric(text) {
			builder.WriteString(text)
			builder.WriteString("\n")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextFromNode(c, builder)
	}
}

// isValidLyric verifica si el texto parece ser una línea de letra válida
func isValidLyric(text string) bool {
	if len(text) < 2 {
		return false
	}

	// Filtrar elementos que no son letras
	invalidPatterns := []string{
		"window.",
		"document.",
		"function()",
		"var ",
		"const ",
		"let ",
		"JSON.parse",
		"<script",
		"</script>",
		"<!--",
	}

	for _, pattern := range invalidPatterns {
		if strings.Contains(text, pattern) {
			return false
		}
	}

	// Debe contener principalmente letras y espacios
	letterCount := 0
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letterCount++
		}
	}

	return float64(letterCount)/float64(len(text)) > 0.3
}

// cleanLyrics limpia y formatea las letras
func cleanLyrics(lyrics string) string {
	// Reemplazar saltos de línea múltiples
	re := regexp.MustCompile(`\n{3,}`)
	lyrics = re.ReplaceAllString(lyrics, "\n\n")

	// Remover espacios al inicio/final de cada línea
	lines := strings.Split(lyrics, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.Join(lines, "\n")
}
