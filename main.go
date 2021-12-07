package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/TheBunnies/TiktokUploaderBot/tiktok"
	"github.com/bwmarrin/discordgo"
)

var rgx = regexp.MustCompile(`http(s|):\/\/.*(tiktok|xzcs3zlph).com.*\/`)

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
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)
	<-sc

	dg.Close()
}
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}
	if rgx.MatchString(m.Content) {
		go func() {
			link := rgx.FindString(m.Content)
			log.Println("Started processing ", link, "Requested by: ", m.Author.Username)
			s.ChannelTyping(m.ChannelID)
			model, err := tiktok.GetDownloadModel(link)
			if err != nil {
				log.Println(err.Error())
				return
			}
			file, err := model.GetConverted()

			if err != nil {
				log.Println(err.Error())
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Sorry, cannot process a video with the following link: %s", link))
				file.Close()
				os.Remove(file.Name())
				return
			}
			_, err = s.ChannelFileSendWithMessage(m.ChannelID, fmt.Sprintf("From: %s \nwith the following message: %s", m.Author.Mention(), rgx.ReplaceAllString(m.Content, "")), model.GetFilename(), file)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" I can't process this video, it's cursed.")
				file.Close()
				os.Remove(file.Name())
				return
			}
			s.ChannelMessageDelete(m.ChannelID, m.Message.ID)
			file.Close()
			os.Remove(file.Name())
		}()
	}
}
