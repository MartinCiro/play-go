# 🎧 Bot de Música - PlayGo

## 📖 Descripción

PlayGo es un bot de música con arquitectura hexagonal que permite reproducir música desde YouTube directamente desde la terminal. Soporta integración con TikTok Live para controlar la reproducción mediante comandos del chat.

## 🚀 Características

- ✅ **Arquitectura hexagonal** - Código mantenible y testeable
- ✅ **Reproducción desde YouTube** - Búsqueda y streaming en tiempo real
- ✅ **Integración con TikTok Live** - Control por comandos del chat
- ✅ **Instalación automática de dependencias** - FFmpeg/ffplay automático
- ✅ **Sistema de logging profesional** - Logs estructurados con niveles
- ✅ **Comandos intuitivos** - Play, skip, queue, revoke

## 🛠️ Instalación

### Prerrequisitos
- **Go 1.21+**
- **Git**

### Configuración inicial

```bash
# Clonar el proyecto
git clone <tu-repositorio>
cd play-go

# Inicializar módulo Go
go mod init github.com/MartinCiro/play-go

# Descargar dependencias
go mod tidy
```

## 🎵 Modo de Uso

### 1. **Bot de Música Standalone (Sin TikTok)**

Ejecuta el bot básico para controlar la música manualmente:

```bash
# Desde la raíz del proyecto
go run cmd/bot/main.go
```

**Comandos disponibles:**
- `!play [canción]` - Añadir canción a la cola
- `!skip` - Saltar canción actual
- `!queue` - Mostrar cola de reproducción  
- `!revoke` - Eliminar tu última canción
- `!exit` - Salir del programa

**Ejemplo:**
```
> !play bad bunny
🎵 Buscando: bad bunny...
✅ Añadido: Bad Bunny - Song Name (Solicitado por: Usuario)

> !queue
🎵 Cola de Reproducción:
   ▶️ 1. Bad Bunny - Song Name
      👤 Usuario
   Total: 1 canciones en cola
```

### 2. **Bot con Integración TikTok Live**

Controla la música mediante comandos del chat de TikTok Live:

```bash
# Conectar a un livestream de TikTok
go run cmd/tiktok-chat/main.go @nombre_usuario_tiktok
```

**Comandos en TikTok Chat:**
- `!play [canción]` - Añadir canción (ej: `!play shakira`)
- `!skip` - Saltar canción actual
- `!queue` - Mostrar cola de reproducción
- `!revoke` - Eliminar tu última canción

**Ejemplo en TikTok Live:**
```
💬 @usuario1: !play despacito
🎵 @usuario1 solicitó: despacito
✅ Añadido: Luis Fonsi - Despacito (Solicitado por: @usuario1)

💬 @usuario2: !skip  
⏭️ @usuario2 solicitó saltar canción
✅ Saltando canción (solicitado por @usuario2)
```

## 🏗️ Estructura del Proyecto

```
play-go/
├── cmd/
│   ├── bot/
│   │   └── main.go                 # Bot standalone
│   └── tiktok-chat/
│       └── main.go                 # Integración TikTok
├── internal/
│   ├── core/                       # Lógica de negocio
│   │   ├── domain/                 # Entidades e interfaces
│   │   ├── application/            # Casos de uso y servicios
│   │   └── usecases/               # Lógica específica
│   └── infrastructure/             # Adaptadores externos
│       ├── adapters/
│       │   ├── providers/          # YouTube, Spotify (futuro)
│       │   ├── player/             # Reproductor ffplay
│       │   └── persistence/        # Almacenamiento en memoria
│       └── delivery/               # CLI y TikTok
├── pkg/
│   ├── ffmpeg/                     # Instalador automático
│   ├── logger/                     # Sistema de logging
│   └── utils/                      # Utilidades compartidas
└── tests/                          # Pruebas unitarias
```

## 🔧 Dependencias Principales

- **`github.com/wader/goutubedl`** - Cliente YouTube
- **`github.com/steampoweredtaco/gotiktoklive`** - Cliente TikTok Live
- **FFmpeg/ffplay** - Reproducción de audio (instalación automática)

## 🐛 Solución de Problemas

### Error de conexión TikTok

- Verifica que el usuario esté en vivo

## 📝 Licencia

Este proyecto es de código abierto para fines educativos. Respeta los términos de servicio de las plataformas integradas.

## 🤝 Contribuciones

Las contribuciones son bienvenidas. Por favor:

1. Fork el proyecto
2. Crea una rama para tu feature
3. Commit tus cambios
4. Push a la rama
5. Abre un Pull Request