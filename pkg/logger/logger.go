package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Logger struct {
	data *strings.Builder
	out  io.Writer
}

func New(out io.Writer) *Logger {
	return &Logger{
		data: &strings.Builder{},
		out:  out,
	}
}

func (l *Logger) Write(data []byte) (int, error) {
	l.data.Write(data)
	return fmt.Fprint(l.out, string(data))
}

// Logger instances
var Log = New(os.Stdout)
var Errors = New(os.Stderr)

// Funciones básicas (sin prefijo)
func Print(a ...any) (int, error) {
	return fmt.Fprint(Log, a...)
}

func Println(a ...any) (int, error) {
	return fmt.Fprintln(Log, a...)
}

func Printf(format string, a ...any) (int, error) {
	return fmt.Fprintf(Log, format, a...)
}

// INFO level - con formato automático (agrega salto de línea)
// Uso: logger.Info("mensaje") -> [INFO] mensaje\n
func Info(a ...any) {
	fmt.Fprint(Log, "[INFO] ")
	fmt.Fprintln(Log, a...)
}

// INFO level - con formato personalizado (agrega salto de línea)
// Uso: logger.Infof("usuario %s conectado", "john") -> [INFO] usuario john conectado\n
func Infof(format string, a ...any) {
	fmt.Fprint(Log, "[INFO] ")
	fmt.Fprintf(Log, format+"\n", a...)
}

// INFO level - sin formato (sin salto de línea)
// Uso: logger.Inf("procesando...") -> [INFO] procesando...
func Inf(a ...any) {
	fmt.Fprint(Log, "[INFO] ")
	fmt.Fprint(Log, a...)
}

// INFO level - con formato personalizado (sin salto de línea)
// Uso: logger.Inff("progreso: %d%%", 50) -> [INFO] progreso: 50%
func Inff(format string, a ...any) {
	fmt.Fprint(Log, "[INFO] ")
	fmt.Fprintf(Log, format, a...)
}

// ERROR level - con formato automático (agrega salto de línea)
// Uso: logger.Error("falló la conexión") -> [ERROR] falló la conexión\n
func Error(a ...any) {
	fmt.Fprint(Errors, "[ERROR] ")
	fmt.Fprintln(Errors, a...)
}

// ERROR level - con formato personalizado (agrega salto de línea)
// Uso: logger.Errorf("error en %s: %v", "función", err) -> [ERROR] error en función: timeout\n
func Errorf(format string, a ...any) {
	fmt.Fprint(Errors, "[ERROR] ")
	fmt.Fprintf(Errors, format+"\n", a...)
}

// ERROR level - sin formato (sin salto de línea)
// Uso: logger.Err("validando...") -> [ERROR] validando...
func Err(a ...any) {
	fmt.Fprint(Errors, "[ERROR] ")
	fmt.Fprint(Errors, a...)
}

// ERROR level - con formato personalizado (sin salto de línea)
// Uso: logger.Errf("intento %d", 3) -> [ERROR] intento 3
func Errf(format string, a ...any) {
	fmt.Fprint(Errors, "[ERROR] ")
	fmt.Fprintf(Errors, format, a...)
}

// FATAL level - con formato automático (agrega salto de línea y termina programa)
// Uso: logger.Fatal("configuración inválida") -> [FATAL] configuración inválida\n + os.Exit(1)
func Fatal(a ...any) {
	fmt.Fprint(Errors, "[FATAL] ")
	fmt.Fprintln(Errors, a...)
	os.Exit(1)
}

// FATAL level - con formato personalizado (agrega salto de línea y termina programa)
// Uso: logger.Fatalf("no se pudo cargar %s", "config.json") -> [FATAL] no se pudo cargar config.json\n + os.Exit(1)
func Fatalf(format string, a ...any) {
	fmt.Fprint(Errors, "[FATAL] ")
	fmt.Fprintf(Errors, format+"\n", a...)
	os.Exit(1)
}

// FATAL level - sin formato (sin salto de línea, termina programa)
// Uso: logger.Fat("abortando") -> [FATAL] abortando + os.Exit(1)
func Fat(a ...any) {
	fmt.Fprint(Errors, "[FATAL] ")
	fmt.Fprint(Errors, a...)
	os.Exit(1)
}

// FATAL level - con formato personalizado (sin salto de línea, termina programa)
// Uso: logger.Fatf("código: %d", 500) -> [FATAL] código: 500 + os.Exit(1)
func Fatf(format string, a ...any) {
	fmt.Fprint(Errors, "[FATAL] ")
	fmt.Fprintf(Errors, format, a...)
	os.Exit(1)
}

// WARN level - con formato personalizado
func Warnf(format string, a ...any) {
	fmt.Fprint(Log, "[WARN] ")
	fmt.Fprintf(Log, format+"\n", a...)
}

// WARN level - automático
func Warn(a ...any) {
	fmt.Fprint(Log, "[WARN] ")
	fmt.Fprintln(Log, a...)
}
