#syntax=docker/dockerfile:1
FROM golang:1.21.5 AS build-stage

WORKDIR /app

RUN apt-get update && apt-get -y install libopus-dev libopusfile-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN ls -al .
RUN go build -o build/bot ./cmd/main.go

RUN ls -al .
RUN ls ./build 

FROM ubuntu:22.04 AS build-release-stage

WORKDIR /

RUN DEBIAN_FRONTEND=noninteractive
RUN apt update && apt upgrade && apt install tzdata
RUN apt-get -y install libopus-dev libopusfile-dev ffmpeg python3 \
    python3-pip && apt-get clean 
RUN pip3 install -U yt-dlp
RUN DEBIAN_FRONTEND=dialog

COPY --from=build-stage /app/build/bot ./
COPY --from=build-stage /app/airhorn.dca ./
COPY --from=build-stage /app/.config.json ./

ENTRYPOINT ["/bot"]

