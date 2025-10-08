package ffmpeg

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/MartinCiro/play-go/pkg/logger"
)

// CheckOrInstall verifica si ffplay está disponible, si no, intenta instalarlo
func CheckOrInstall() error {
	if isFFPlayAvailable() {
		logger.Info("✅ ffplay encontrado")
		return nil
	}

	logger.Warn("❌ ffplay no encontrado")
	logger.Info("🔄 Intentando instalar ffmpeg...")

	switch runtime.GOOS {
	case "windows":
		return installWindows()
	case "darwin":
		return installMacOS()
	case "linux":
		return installLinux()
	default:
		return fmt.Errorf("sistema operativo no soportado: %s", runtime.GOOS)
	}
}

// isFFPlayAvailable verifica si ffplay está disponible
func isFFPlayAvailable() bool {
	cmd := exec.Command("ffplay", "-version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// installWindows instala ffmpeg en Windows
func installWindows() error {
	logger.Info("📥 Instalando ffmpeg en Windows...")

	// Método 1: Usar winget (Windows 10+)
	if isCommandAvailable("winget") {
		logger.Info("🔍 Usando winget para instalar ffmpeg...")
		cmd := exec.Command("winget", "install", "Gyan.FFmpeg")
		if output, err := cmd.CombinedOutput(); err == nil {
			logger.Info("✅ ffmpeg instalado exitosamente con winget")
			return nil
		} else {
			logger.Warnf("❌ winget falló: %s", string(output))
		}
	}

	// Método 2: Métodos alternativos
	return downloadWindowsFFmpeg()
}

// downloadWindowsFFmpeg instala ffmpeg en Windows con múltiples métodos
func downloadWindowsFFmpeg() error {
	logger.Info("📥 Intentando métodos alternativos para Windows...")

	// Método 1: Usar Python pip (si está disponible)
	if isCommandAvailable("python") || isCommandAvailable("python3") {
		logger.Info("🔍 Intentando instalar con pip...")
		pythonCmd := "python"
		if !isCommandAvailable("python") {
			pythonCmd = "python3"
		}

		cmd := exec.Command(pythonCmd, "-m", "pip", "install", "ffmpeg-python")
		if output, err := cmd.CombinedOutput(); err == nil {
			logger.Info("✅ ffmpeg instalado exitosamente con pip")
			return nil
		} else {
			logger.Warnf("❌ pip falló: %s", string(output))
		}
	}

	// Método 2: Usar Chocolatey
	if isCommandAvailable("choco") {
		logger.Info("🔍 Intentando instalar con Chocolatey...")
		cmd := exec.Command("choco", "install", "ffmpeg", "-y")
		if output, err := cmd.CombinedOutput(); err == nil {
			logger.Info("✅ ffmpeg instalado exitosamente con Chocolatey")
			return nil
		} else {
			logger.Warnf("❌ Chocolatey falló: %s", string(output))
		}
	}

	// Método 3: Usar Scoop
	if isCommandAvailable("scoop") {
		logger.Info("🔍 Intentando instalar con Scoop...")
		cmd := exec.Command("scoop", "install", "ffmpeg")
		if output, err := cmd.CombinedOutput(); err == nil {
			logger.Info("✅ ffmpeg instalado exitosamente con Scoop")
			return nil
		} else {
			logger.Warnf("❌ Scoop falló: %s", string(output))
		}
	}

	// Método 4: Descarga directa con instrucciones específicas
	return downloadWindowsManual()
}

// downloadWindowsManual guía al usuario para instalación manual en Windows
func downloadWindowsManual() error {
	logger.Error("❌ No se pudo instalar ffmpeg automáticamente")
	logger.Info("💡 Por favor, instala ffmpeg manualmente con uno de estos métodos:")
	logger.Info("")
	logger.Info("📦 MÉTODO 1 - Chocolatey (Recomendado):")
	logger.Info("   Abre PowerShell como Administrador y ejecuta:")
	logger.Info("   Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))")
	logger.Info("   choco install ffmpeg -y")
	logger.Info("")
	logger.Info("📦 MÉTODO 2 - Winget (Windows 10+):")
	logger.Info("   Abre PowerShell y ejecuta:")
	logger.Info("   winget install Gyan.FFmpeg")
	logger.Info("")
	logger.Info("📦 MÉTODO 3 - Descarga manual:")
	logger.Info("   1. Ve a: https://github.com/BtbN/FFmpeg-Builds/releases")
	logger.Info("   2. Descarga 'ffmpeg-master-latest-win64-gpl.zip'")
	logger.Info("   3. Extrae la carpeta 'bin' a C:\\ffmpeg\\")
	logger.Info("   4. Agrega C:\\ffmpeg\\bin al PATH de Windows")
	logger.Info("   5. Reinicia la terminal")
	logger.Info("")
	logger.Info("🔄 Después de instalar, ejecuta este programa nuevamente")

	return fmt.Errorf("instalación manual requerida")
}

// installMacOS instala ffmpeg en macOS
func installMacOS() error {
	logger.Info("📥 Instalando ffmpeg en macOS...")

	// Método 1: Usar Homebrew
	if isCommandAvailable("brew") {
		logger.Info("🔍 Usando Homebrew para instalar ffmpeg...")
		cmd := exec.Command("brew", "install", "ffmpeg")
		if output, err := cmd.CombinedOutput(); err == nil {
			logger.Info("✅ ffmpeg instalado exitosamente con Homebrew")
			return nil
		} else {
			logger.Warnf("❌ Homebrew falló: %s", string(output))
		}
	}

	// Método 2: Usar MacPorts
	if isCommandAvailable("port") {
		logger.Info("🔍 Usando MacPorts para instalar ffmpeg...")
		cmd := exec.Command("sudo", "port", "install", "ffmpeg")
		if output, err := cmd.CombinedOutput(); err == nil {
			logger.Info("✅ ffmpeg instalado exitosamente con MacPorts")
			return nil
		} else {
			logger.Warnf("❌ MacPorts falló: %s", string(output))
		}
	}

	// Método 3: Instrucciones manuales
	logger.Error("❌ No se pudo instalar ffmpeg automáticamente")
	logger.Info("💡 Por favor, instala ffmpeg manualmente:")
	logger.Info("")
	logger.Info("📦 MÉTODO 1 - Homebrew:")
	logger.Info("   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
	logger.Info("   brew install ffmpeg")
	logger.Info("")
	logger.Info("📦 MÉTODO 2 - MacPorts:")
	logger.Info("   Visita: https://www.macports.org/install.php")
	logger.Info("   sudo port install ffmpeg")
	logger.Info("")
	logger.Info("📦 MÉTODO 3 - Descarga manual:")
	logger.Info("   Visita: https://evermeet.cx/ffmpeg/")
	logger.Info("")
	logger.Info("🔄 Después de instalar, ejecuta este programa nuevamente")

	return fmt.Errorf("instalación manual requerida")
}

// installLinux mejora con comandos específicos
func installLinux() error {
	logger.Info("📥 Instalando ffmpeg en Linux...")

	// Detectar distribución y gestor de paquetes
	distro := detectLinuxDistro()
	logger.Infof("🔍 Distribución detectada: %s", distro)

	commands := []struct {
		name    string
		cmd     string
		args    []string
		sudo    bool
		distros []string
	}{
		// Ubuntu/Debian
		{
			name:    "APT",
			cmd:     "apt",
			args:    []string{"install", "-y", "ffmpeg"},
			sudo:    true,
			distros: []string{"ubuntu", "debian"},
		},
		// Fedora
		{
			name:    "DNF",
			cmd:     "dnf",
			args:    []string{"install", "-y", "ffmpeg"},
			sudo:    true,
			distros: []string{"fedora"},
		},
		// CentOS/RHEL
		{
			name:    "YUM",
			cmd:     "yum",
			args:    []string{"install", "-y", "ffmpeg"},
			sudo:    true,
			distros: []string{"centos", "rhel"},
		},
		// Arch Linux
		{
			name:    "Pacman",
			cmd:     "pacman",
			args:    []string{"-S", "--noconfirm", "ffmpeg"},
			sudo:    true,
			distros: []string{"arch"},
		},
		// openSUSE
		{
			name:    "Zypper",
			cmd:     "zypper",
			args:    []string{"install", "-y", "ffmpeg"},
			sudo:    true,
			distros: []string{"opensuse"},
		},
		// Snap (universal)
		{
			name:    "Snap",
			cmd:     "snap",
			args:    []string{"install", "ffmpeg"},
			sudo:    false,
			distros: []string{"ubuntu", "debian", "fedora", "centos", "arch", "opensuse"},
		},
		// Flatpak (universal)
		{
			name:    "Flatpak",
			cmd:     "flatpak",
			args:    []string{"install", "flathub", "org.freedesktop.Platform.ffmpeg", "-y"},
			sudo:    false,
			distros: []string{"ubuntu", "debian", "fedora", "centos", "arch", "opensuse"},
		},
	}

	for _, pm := range commands {
		// Verificar si es compatible con la distribución
		if !contains(pm.distros, distro) && !contains(pm.distros, "all") {
			continue
		}

		if isCommandAvailable(pm.cmd) {
			logger.Infof("🔍 Intentando instalar con %s...", pm.name)

			var cmd *exec.Cmd
			if pm.sudo {
				cmd = exec.Command("sudo", append([]string{pm.cmd}, pm.args...)...)
			} else {
				cmd = exec.Command(pm.cmd, pm.args...)
			}

			if output, err := cmd.CombinedOutput(); err == nil {
				logger.Infof("✅ ffmpeg instalado exitosamente con %s", pm.name)
				return nil
			} else {
				logger.Warnf("❌ %s falló: %s", pm.name, string(output))
			}
		}
	}

	// Si todo falla, mostrar comandos específicos
	return showLinuxManualInstructions(distro)
}

// detectLinuxDistro detecta la distribución de Linux
func detectLinuxDistro() string {
	// Leer /etc/os-release
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, `"`)
			return id
		}
	}
	return "unknown"
}

// showLinuxManualInstructions muestra instrucciones específicas para cada distro
func showLinuxManualInstructions(distro string) error {
	logger.Error("❌ No se pudo instalar ffmpeg automáticamente")
	logger.Info("💡 Por favor, instala ffmpeg manualmente:")

	switch distro {
	case "ubuntu", "debian":
		logger.Info("   sudo apt update && sudo apt install ffmpeg")
	case "fedora":
		logger.Info("   sudo dnf install ffmpeg")
	case "centos", "rhel":
		logger.Info("   sudo yum install epel-release && sudo yum install ffmpeg")
	case "arch":
		logger.Info("   sudo pacman -S ffmpeg")
	case "opensuse":
		logger.Info("   sudo zypper install ffmpeg")
	default:
		logger.Info("   Consulta: https://ffmpeg.org/download.html")
	}

	logger.Info("🔄 Después de instalar, ejecuta este programa nuevamente")
	return fmt.Errorf("instalación manual requerida")
}

// isCommandAvailable verifica si un comando está disponible
func isCommandAvailable(name string) bool {
	cmd := exec.Command("which", name)
	if runtime.GOOS == "windows" {
		cmd = exec.Command("where", name)
	}
	return cmd.Run() == nil
}

// GetFFPlayCommand devuelve el comando ffplay con verificación
func GetFFPlayCommand() (string, error) {
	if !isFFPlayAvailable() {
		return "", fmt.Errorf("ffplay no disponible")
	}
	return "ffplay", nil
}

// contains verifica si un string está en un slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
