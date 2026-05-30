# Caffeine Adct Discord Bot

A music-playing Discord bot built in Go. Streams audio from YouTube and other supported sites directly to a Discord voice channel, with a playlist queue, playback controls, and a built-in airhorn sound effect.

## Features

- **Music playback** — Add songs from YouTube and any site supported by [yt-dlp](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md)
- **Playlist queue** — Queue multiple songs; the bot loops back to the start when the queue ends
- **Playback controls** — Play, pause, skip to any position in the queue
- **Airhorn** — Instant airhorn sound effect on demand
- **Redis caching** — Encoded audio is cached for 10 minutes to avoid re-downloading repeat songs
- **Docker support** — Ready-to-run with Docker Compose (bot + Redis)
- **Multi-channel & multi-server** — Separate independent music players per guild and per voice channel; play different songs in different channels simultaneously

## Prerequisites

- [Go](https://go.dev/) 1.21.5+ (if not using Docker)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [ffmpeg](https://ffmpeg.org/)
- [Redis](https://redis.io/) (or Docker for the Redis container)

## Setup

### 1. Create a Discord application

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application, then go to the **Bot** tab
3. Click **Reset Token** and copy the token
4. Enable **Message Content Intent** under Privileged Gateway Intents
5. Invite the bot to your server using the OAuth2 URL generator (scopes: `bot`, permissions: `Send Messages`, `Connect`, `Speak`)

### 2. Configure the bot

Create a `config.json` in the project root:

```json
{
    "discord_app_key": "YOUR_DISCORD_BOT_TOKEN",
    "redis_url": "redis:6379"
}
```

### 3. Run

#### With Go directly
```bash
go run ./cmd/main.go
```

#### With Make
```bash
make run
```

#### With Docker Compose
```bash
docker compose up
```

The Makefile provides additional targets:

| Target     | Description                        |
|------------|------------------------------------|
| `make`     | Check for ffmpeg and yt-dlp        |
| `make run` | Check deps, build, and run the bot |
| `make build/bot` | Build binary to `./build/bot` |

## Commands

| Command             | Description                              |
|---------------------|------------------------------------------|
| `wassup`            | Reply with a greeting                    |
| `join`              | Join the user's current voice channel    |
| `add <url>`         | Add a song from YouTube (or other site) to the queue |
| `play`              | Start or resume playback                |
| `pause`             | Toggle pause on the current song        |
| `list`              | Show the current queue with indices     |
| `switch <number>`   | Skip to the song at the given position  |
| `!airhorn`          | Play the airhorn sound effect           |

## Architecture

The audio playback pipeline works as follows:

```
User sends URL  →  yt-dlp downloads audio to stdout
                         ↓
                    ffmpeg converts to raw PCM (s16le, 48kHz)
                         ↓
              Go Opus encoder encodes PCM → Opus packets
                         ↓
              Opus packets sent over Discord voice WebSocket
```

Downloaded and encoded audio is cached in Redis for 10 minutes so repeated plays of the same song avoid re-downloading.

## Project Structure

```
├── cmd/main.go                 # Entry point: Redis init, bot start, audio streaming
├── config/
│   └── config.go               # Loads discord_app_key from config.json
├── internal/
│   ├── audio/
│   │   ├── init.go             # Loads airhorn.dca into memory
│   │   └── sound.go            # OpusSound type, PlaySoundToVC, DCA parsing
│   ├── bot/
│   │   ├── bot.go              # Command routing, message handlers
│   │   └── type.go             # DisBot struct and ActionType constants
│   ├── cache/
│   │   └── cache.go            # Cache interface definition
│   ├── cache/rediscache/
│   │   └── redis.go            # Redis-backed cache implementation
│   └── musicplayer/
│       └── music_player.go     # Queue management, play/pause/switch
├── utils/
│   ├── utils.go                # Helpers: PrintJSONs, GetTYVidIDFromURL
│   └── ytutils/
│       └── ytdlp.go            # yt-dlp metadata fetch
├── airhorn.dca                 # Pre-encoded Opus audio for the airhorn
├── config.json                 # Discord bot token (user-supplied)
├── dockerfile                  # Multi-stage Docker build
├── compose.yaml                # Docker Compose: bot + redis
├── Makefile                    # Build and run targets
├── go.mod / go.sum             # Go module and dependencies
```

## License

This project is developed for personal use.
