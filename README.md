# About 
This bot opens full potential of HTML5 video embeds, it makes an HTTP request to the tiktok API and sends it directly to your discord chat!

If it detects any shared Tiktok URL on your discord server, the sent message is going to be deleted and replaced with video embed as soon as possible.

## Running bot on your Windows PC
1. Install GO runtime from the [official website](https://go.dev/).
2. Open **config.json** and pass down your discord bot token.
3. Run `go run main.go` or build `go build .`

## Proxying
This bot supports proxying. Supply them in **config.json** file or leave them empty if not needed.

## FFmpeg
This bot has a ffmpeg dependency. Be sure to install it if running locally.

## Linux and Docker support
1. Install [Docker](https://www.docker.com/) on your main OS.
2. Build docker image `docker build -t discord-bot .`
3. Run the container `docker run --rm -d discord-bot`