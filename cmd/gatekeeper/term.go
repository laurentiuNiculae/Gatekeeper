package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tsoding/gatekeeper/internal"
	"github.com/bwmarrin/discordgo"
)

type Message struct {
	ID        string
	ChannelID string
	Content   string
	Timestamp time.Time
	Author    User
	// GuildID           string
	// EditedTimestamp   *time.Time
	// MentionRoles      []string
	// TTS               bool
	// MentionEveryone   bool
	// Attachments       []*discordgo.MessageAttachment
	// Components        []discordgo.MessageComponent
	// Embeds            []*discordgo.MessageEmbed
	// Mentions          []User
	// Reactions         []*discordgo.MessageReactions
	// Pinned            bool
	// Type              discordgo.MessageType
	// WebhookID         string
	// Member            *discordgo.Member
	// MentionChannels   []*discordgo.Channel
	// Activity          *discordgo.MessageActivity
	// Application       *discordgo.MessageApplication
	// MessageReference  *discordgo.MessageReference
	// ReferencedMessage *discordgo.Message
	// Interaction       *discordgo.MessageInteraction
	// Flags             discordgo.MessageFlags
	// Thread            *discordgo.Channel
	// StickerItems      []*discordgo.Sticker
}

type MockDiscordSession struct {
}

func (m *MockDiscordSession) ChannelMessageSend(channelID string, content string) (*discordgo.Message, error) {
	log.Printf("Channel: %s, Message: %s\n", channelID, content)

	return nil, nil
}

type User struct {
	ID       string
	Email    string
	Username string
	Bot      bool
}

type LocalEnvironment struct {
	m Message
}

func (l *LocalEnvironment) AtAdmin() string {
	return "<@Admin>"
}

func (l *LocalEnvironment) AtAuthor() string {
	return fmt.Sprintf("<@%s>", l.m.Author.ID)
}

func (l *LocalEnvironment) AuthorUserId() string {
	return l.m.Author.ID
}

func (l *LocalEnvironment) IsAuthorAdmin() bool {
	return l.m.Author.ID == "Admin"
}

func (l *LocalEnvironment) AsDiscord() *DiscordEnvironment {
	return &DiscordEnvironment{}
}

func (l *LocalEnvironment) SendMessage(message string) {
	fmt.Println("Bot said:\n" + message)
}

func logLocalMessage(db *sql.DB, m Message) {
	_, err := db.Exec("INSERT INTO Discord_Log (message_id, user_id, user_name, text) VALUES ($1, $2, $3, $4)", m.ID, m.Author.ID, m.Author.Username, m.Content)
	if err != nil {
		log.Println("ERROR: logDiscordMessage: could not insert element", m.Author.ID, m.Author.Username, m.Content, ":", err)
		return
	}
}

func handleMessages(db *sql.DB, m Message) {
	if m.Author.Bot {
		return
	}

	logLocalMessage(db, m)

	command, ok := parseCommand(m.Content)
	if !ok {
		if db != nil {
			internal.FeedMessageToCarrotson(db, m.Content)
		}
		return
	}

	env := &LocalEnvironment{
		m: m,
	}
	EvalCommand(db, command, env)
}

type CmdListener struct {
	registered []chan<- string
}

func (cl *CmdListener) register() <-chan string {
	ch := make(chan string, 10)
	cl.registered = append(cl.registered, ch)

	return ch
}

func (cl *CmdListener) handle(f func(message string)) {
	go func() {
		ch := cl.register()

		for line := range ch {
			f(line)
		}
	}()
}

func (cl *CmdListener) start() {
	fmt.Println("Bot Started in CLI mode.")
	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(line)

			for _, ch := range cl.registered {
				select {
				case ch <- line:
				default:
				}
			}
		}
	}()
}

func startCLI(db *sql.DB) error {
	cmdListener := &CmdListener{}

	cmdListener.handle(func(message string) {
		handleMessages(db, Message{
			ID:        "",
			ChannelID: "",
			Content:   message,
			Timestamp: time.Now(),
			Author: User{
				ID:       "Admin",
				Email:    "",
				Username: "",
				Bot:      false,
			},
		})
	})

	cmdListener.start()

	return nil
}
