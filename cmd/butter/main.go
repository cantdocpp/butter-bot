package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

var (
	discord       *discordgo.Session
	yt            *youtube.Client
	encodeOptions *dca.EncodeOptions
)

const (
	BOT_PREFIX = "butter"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	var err error
	discord, err = discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	yt = &youtube.Client{}
	encodeOptions = dca.StdEncodeOptions
	encodeOptions.RawOutput = true
	encodeOptions.Bitrate = 24
	encodeOptions.Application = "lowdelay"
}

func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	mSplit := strings.Split(m.Content, " ")
	if mSplit[0] != BOT_PREFIX {
		return
	}

	if len(mSplit) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Please specify the command")
		return
	}

	command := mSplit[1]
	// fmt.Println(command)

	switch command {
	case "play":
		if len(mSplit) < 3 {
			s.ChannelMessageSend(m.ChannelID, "Please specify the url")
			return
		}

		channel, err := s.State.Channel(m.ChannelID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		guild, err := s.State.Guild(channel.GuildID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		channelID := ""
		for _, vs := range guild.VoiceStates {
			if vs.UserID == m.Author.ID {
				channelID = vs.ChannelID
				break
			}
		}

		if channelID == "" {
			s.ChannelMessageSend(m.ChannelID, "You aren't in a voice channel")
			return
		}

		ytUrl := mSplit[2]
		url, err := getStreamUrl(ytUrl)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		encodingSession, err := dca.EncodeFile(url, encodeOptions)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		defer encodingSession.Cleanup()

		vc, err := s.ChannelVoiceJoin(guild.ID, channelID, false, false)
		if err != nil {
			if _, ok := s.VoiceConnections[guild.ID]; ok {
				vc = s.VoiceConnections[guild.ID]
			} else {
				s.ChannelMessageSend(m.ChannelID, err.Error())
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}

		vc.Speaking(true)
		done := make(chan error)
		dca.NewStream(encodingSession, vc, done)
		err = <-done
		if err != nil && err != io.EOF {
			s.ChannelMessageSend(m.ChannelID, err.Error())
		}
		vc.Speaking(false)
		vc.Disconnect()
	default:
		s.ChannelMessageSend(m.ChannelID, "No command found")
	}
}

func init() {
	discord.AddHandler(MessageHandler)
}

func main() {
	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot is up!")
	})

	err := discord.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer discord.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	fmt.Println("Bot is down...")
}
