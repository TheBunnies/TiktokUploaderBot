# About 
This bot opens full potential of HTML5 video embeds, it makes an HTTP request to https://www.tiktokdownloader.org/ and converts the data to x254 codec using FFmpeg so that discord could handle it properly.

If it detects any shared Tiktok URL on your discord server, the sent message is going to be deleted and replaced with video embed as soon as possible.

## Running bot on your Windows PC
1. Install GO runtime from the [official website](https://go.dev/).
2. Download [FFmpeg](http://www.ffmpeg.org/) and drag an executable to the root folder of your project.
3. Open **config.json** and pass down your discord bot token.
3. Run `go run main.go` or build `go build .`

## Linux and Docker support
1. Install [Docker](https://www.docker.com/) on your main OS.
2. Build docker image `docker build -t discord-bot .`
3. Run the container `docker run --rm -d discord-bot`