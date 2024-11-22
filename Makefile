FFMPEG = ffmpeg
YTDLP = yt-dlp

all: check_ffmpeg check_ytdlp

check_ffmpeg:
	@command -v ${FFMPEG} >/dev/null 2>&1 || (echo "Installing ${FFMPEG}..." && apt update && apt upgrade && apt install)

check_ytdlp:
	@command -v ${YTDLP} >/dev/null 2>&1 || (echo "Installing ${YTDLP}..." && apt update && apt upgrade && apt install)

run: build/bot check_ffmpeg check_ytdlp
	./build/bot	

build build/bot:
	go build -o ./build/bot cmd/main.go 