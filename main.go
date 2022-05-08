package main

import (
	"encoding/json"
	"fmt"
	"github.com/TheBunnies/TiktokUploaderBot/tiktok"
	"github.com/TheBunnies/TiktokUploaderBot/twitter"
	"github.com/servusdei2018/shards"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Token string `json:"token"`
	Proxy struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Ip       string `json:"ip"`
		Port     string `json:"port"`
	}
}

var (
	rgxTiktok  = regexp.MustCompile(`http(s|):\/\/.*(tiktok).com[^\s]*`)
	rgxTwitter = regexp.MustCompile(`http(|s):\/\/twitter\.com\/i\/status\/[0-9]*`)
	roleRgx    = regexp.MustCompile(`<@&(\d+)>`)

	ConfigBody Config

	Dg *shards.Manager
)

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err.Error())
	}
	ConfigBody = config
	setupBot(config.Token)
}

func loadConfig() (Config, error) {
	if _, err := os.Stat("config.json"); err != nil {
		os.Create("config.json")
		file, err := os.OpenFile("config.json", os.O_APPEND, os.ModeAppend)
		if err != nil {
			return Config{}, err
		}
		defer file.Close()
		token := Config{
			Token: "your token goes here",
			Proxy: struct {
				User     string `json:"user"`
				Password string `json:"password"`
				Ip       string `json:"ip"`
				Port     string `json:"port"`
			}{User: "user", Password: "password", Ip: "ip", Port: "port"},
		}
		err = json.NewEncoder(file).Encode(token)
		if err != nil {
			return Config{}, err
		}
		return Config{}, err
	}
	file, err := os.Open("config.json")
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	config := Config{}
	json.NewDecoder(file).Decode(&config)
	return config, nil

}
func setupBot(token string) {
	Dg, err := shards.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	Dg.AddHandler(messageCreate)
	Dg.AddHandler(onConnect)

	Dg.RegisterIntent(discordgo.IntentsGuildMessages)

	err = Dg.Start()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	Dg.Shutdown()
}
func onConnect(s *discordgo.Session, evt *discordgo.Connect) {
	log.Printf("Shard #%v connected.\n", s.ShardID)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}
	checkTiktok(s, m)
	checkTwitter(s, m)
}

func checkTwitter(s *discordgo.Session, m *discordgo.MessageCreate) {
	if rgxTwitter.MatchString(m.Content) {
		link := TrimURL(rgxTwitter.FindString(m.Content))
		log.Println("Started processing twitter request", link, "Requested by:", m.Author.Username, ":", m.Author.ID)
		s.ChannelTyping(m.ChannelID)

		data := twitter.NewTwitterVideoDownloader(link)
		proxy := "http://" + ConfigBody.Proxy.User + ":" + ConfigBody.Proxy.Password + "@" + ConfigBody.Proxy.Ip + ":" + ConfigBody.Proxy.Port

		guild, _ := s.Guild(m.GuildID)
		limit := getDownloadSizeLimit(guild)
		file, err := data.Download(proxy, limit)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Sorry could not get the details about the video `%s`", err))
			return
		}
		var message string
		if rgxTwitter.ReplaceAllString(m.Content, "") == "" {
			message = fmt.Sprintf("From: %s \nOriginal link: <%s>", m.Author.Mention(), link)
		} else {
			content := removeRoleMentions(strings.TrimSpace(rgxTwitter.ReplaceAllString(m.Content, "")))
			message = fmt.Sprintf("From: %s \nOriginal link: <%s> \nWith the following message: %s", m.Author.Mention(), link, content)
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
func checkTiktok(s *discordgo.Session, m *discordgo.MessageCreate) {
	if rgxTiktok.MatchString(m.Content) {
		link := TrimURL(rgxTiktok.FindString(m.Content))
		log.Println("Started processing tiktok request", link, "Requested by:", m.Author.Username, ":", m.Author.ID)
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

		proxy := "http://" + ConfigBody.Proxy.User + ":" + ConfigBody.Proxy.Password + "@" + ConfigBody.Proxy.Ip + ":" + ConfigBody.Proxy.Port
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			log.Println(err)
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		data, err := tiktok.NewAwemeDetail(parsedId, transport)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s It looks like I can't get the details about your video, try to resend it one more time!", m.Author.Mention()))
			return
		}
		guild, _ := s.Guild(m.GuildID)
		limit := getDownloadSizeLimit(guild)
		file, err := data.DownloadVideo(limit)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s Sorry, cannot process the video `%s`", m.Author.Mention(), err))
			return
		}
		var message string
		if rgxTiktok.ReplaceAllString(m.Content, "") == "" {
			message = fmt.Sprintf("From: %s \nAuthor: **%s** \nDuration: `%s`\nCreation time: `%s`\nDescription: ||%s|| \nOriginal link: <%s>", m.Author.Mention(), data.Author.Unique_ID, data.Duration(), data.Time(), data.Description(), link)
		} else {
			content := removeRoleMentions(strings.TrimSpace(rgxTiktok.ReplaceAllString(m.Content, "")))
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
	loc.Scheme = "https"
	return loc.String()
}
func removeRoleMentions(message string) string {
	m := roleRgx.ReplaceAllString(message, "**<REDACTED MENTION>**")
	return strings.Replace(m, "@everyone", "**<REDACTED MENTION>**", -1)
}
