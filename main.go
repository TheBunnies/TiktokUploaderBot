package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/TheBunnies/TiktokUploaderBot/tiktok"
	"github.com/bwmarrin/discordgo"
)

var rgx = regexp.MustCompile(`http(s|):\/\/.*(tiktok).com[^\s]*`)
var roleRgx = regexp.MustCompile(`<@&(\d+)>`)

type Token struct {
	Body string `json:"token"`
}

func main() {
	token, err := loadConfig()
	if err != nil {
		log.Fatal(err.Error())
	}
	setupBot(token)
}
func loadConfig() (string, error) {
	if _, err := os.Stat("config.json"); err != nil {
		os.Create("config.json")
		file, err := os.OpenFile("config.json", os.O_APPEND, os.ModeAppend)
		if err != nil {
			return "", err
		}
		defer file.Close()
		token := Token{
			Body: "your token goes here",
		}
		err = json.NewEncoder(file).Encode(token)
		if err != nil {
			return "", err
		}
		return "", err
	}
	file, err := os.Open("config.json")
	if err != nil {
		return "", err
	}
	defer file.Close()
	token := Token{}
	json.NewDecoder(file).Decode(&token)
	return token.Body, nil

}
func setupBot(token string) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}
	if rgx.MatchString(m.Content) {
		link := TrimURL(rgx.FindString(m.Content))
		log.Println("Started processing ", link, "Requested by:", m.Author.Username, ":", m.Author.ID)
		s.ChannelTyping(m.ChannelID)

		id, err := tiktok.GetId(link)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s Sorry, cannot access the link to your video `%s`", m.Author.Mention(), err))
			return
		}
		parsedId, err := tiktok.Parse(id)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s Sorry, could not parse the actual id of the video", m.Author.Mention()))
			return
		}
		data, err := tiktok.NewAwemeDetail(parsedId)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s It looks like I can't get the details about your video, try to resend it one more time!", m.Author.Mention()))
			return
		}
		guild, _ := s.Guild(m.GuildID)
		limit := getDownloadSizeLimit(guild)
		file, err := tiktok.DownloadVideo(data, limit)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s Sorry, cannot process the video `%s`", m.Author.Mention(), err))
			return
		}
		var message string
		if rgx.ReplaceAllString(m.Content, "") == "" {
			message = fmt.Sprintf("From: %s \nAuthor: **%s** \nDuration: `%s`\nCreation time: `%s`\nDescription: ||%s|| \nOriginal link: <%s>", m.Author.Mention(), data.Author.Unique_ID, data.Duration(), data.Time(), data.Description(), link)
		} else {
			content := removeRoleMentions(strings.TrimSpace(rgx.ReplaceAllString(m.Content, "")))
			message = fmt.Sprintf("From: %s\nAuthor: **%s** \nDuration: `%s`\nCreation time: `%s`\nDescription: ||%s|| \nOriginal link: <%s> \nwith the following message: %s", m.Author.Mention(), data.Author.Unique_ID, data.Duration(), data.Time(), data.Description(), link, content)
		}
		_, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{File: &discordgo.File{Name: file.Name(), ContentType: "video/mp4", Reader: file}, Content: message, Reference: m.MessageReference})
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" I can't process this video, it's cursed.")
			file.Close()
			os.Remove(file.Name())
			return
		}
		s.ChannelMessageDelete(m.ChannelID, m.Message.ID)
		file.Close()
		os.Remove(file.Name())
	}
}
func getDownloadSizeLimit(guild *discordgo.Guild) int64 {
	tier := guild.PremiumTier
	if tier == discordgo.PremiumTier2 {
		return 50000000
	} else if tier == discordgo.PremiumTier3 {
		return 100000000
	}
	return 8000000
}

func TrimURL(uri string) string {
	loc, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	loc.RawQuery = ""
	loc.Scheme = "http"
	return loc.String()
}
func removeRoleMentions(message string) string {
	m := roleRgx.ReplaceAllString(message, "**<REDACTED MENTION>**")
	return strings.Replace(m, "@everyone", "**<REDACTED MENTION>**", -1)
}
