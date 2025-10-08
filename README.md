# 🎧 Bot de música para integrar con APIs

## 🚀 Iniciar proyecto

```bash
go mod tidy
```

## 🆕 Inicializar módulo Go

```bash
go mod init
```

## 📦 Obtener paquetes

```bash
go get github.com/wader/goutubedl
```

## 📂 Estructura del proyecto

* `cmd` para el punto de entrada.
* `internal` para la lógica de negocio, infraestructura y casos de uso.
* `pkg` para librerías reutilizables.
* `tests` para pruebas unitarias e integraciones.

```
play-go/
├── cmd/
│   └── bot/
│       └── main.go                 # Punto de entrada de la aplicación
├── internal/
│   ├── core/
│   │   ├── domain/
│   │   │   ├── song.go             # Entidad principal Song
│   │   │   ├── playlist.go         # Lógica de la playlist
│   │   │   ├── player.go           # Interfaces del reproductor
│   │   │   └── provider.go         # Interfaces de proveedores de música
│   │   ├── application/
│   │   │   ├── service.go          # MusicService - orquestador principal
│   │   │   ├── commands/           # Handlers de comandos
│   │   │   │   ├── play.go
│   │   │   │   ├── skip.go
│   │   │   │   ├── revoke.go
│   │   │   │   └── queue.go
│   │   │   └── ports/              # Puertos (interfaces)
│   │   │       ├── player_port.go
│   │   │       ├── provider_port.go
│   │   │       └── repository_port.go
│   │   └── usecases/               # Casos de uso específicos
│   │       ├── play_song.go
│   │       ├── manage_playlist.go
│   │       └── player_control.go
│   └── infrastructure/
│       ├── adapters/
│       │   ├── providers/          # Adaptadores de proveedores de música
│       │   │   ├── youtube/
│       │   │   │   ├── youtube_provider.go
│       │   │   │   └── youtube_client.go
│       │   │   ├── spotify/        # Para futura integración
│       │   │   │   └── spotify_provider.go
│       │   │   └── soundcloud/     # Para futura integración
│       │   │       └── soundcloud_provider.go
│       │   ├── player/
│       │   │   └── ffplay/
│       │   │       ├── ffplay_player.go
│       │   │       └── audio_stream.go
│       │   └── persistence/
│       │       ├── memory/         # Implementación en memoria
│       │       │   ├── playlist_repository.go
│       │       │   └── state_repository.go
│       │       └── redis/          # Para persistencia futura
│       │           └── playlist_repository.go
│       ├── delivery/
│       │   ├── cli/                # Capa de presentación CLI
│       │   │   ├── handler.go
│       │   │   ├── parser.go
│       │   │   └── presenter.go
│       │   └── discord/            # Para futura integración con Discord
│       │       ├── bot.go
│       │       └── command_handler.go
│       └── config/
│           ├── config.go
│           └── providers.go        # Configuración de proveedores
├── pkg/
│   ├── audio/                      # Utilidades de audio compartidas
│   │   ├── stream.go
│   │   └── format.go
│   └── utils/
│       ├── logger.go
│       └── helpers.go
├── tests/
│   ├── unit/
│   │   ├── core/
│   │   ├── application/
│   │   └── infrastructure/
│   ├── integration/
│   │   ├── providers/
│   │   └── player/
│   └── mocks/
│       ├── player_mock.go
│       ├── provider_mock.go
│       └── repository_mock.go
├── go.mod
├── go.sum
├── Makefile
├── docker-compose.yml              # Para dependencias (Redis, etc.)
└── README.md
```
