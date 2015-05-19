package slackbot

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	EventTypeMessage                    EventType = "message"
	EventTypeHello                      EventType = "hello"
	EventTypeChannelMarked              EventType = "channel_marked"
	EventTypeChannelCreated             EventType = "channel_created"
	EventTypeChannelJoined              EventType = "channel_joined"
	EventTypeChannelLeft                EventType = "channel_left"
	EventTypeChannelDeleted             EventType = "channel_deleted"
	EventTypeChannelRename              EventType = "channel_rename"
	EventTypeChannelArchive             EventType = "channel_archive"
	EventTypeChannelUnarchive           EventType = "channel_unarchive"
	EventTypeChannelHistoryChanged      EventType = "channel_history_changed"
	EventTypeChannelImCreated           EventType = "im_created"
	EventTypeChannelImOpen              EventType = "im_open"
	EventTypeChannelImClose             EventType = "im_close"
	EventTypeChannelImMarked            EventType = "im_marked"
	EventTypeChannelImHistoryChanged    EventType = "im_history_changed"
	EventTypeChannelGroupJoined         EventType = "group_joined"
	EventTypeChannelGroupLeft           EventType = "group_left"
	EventTypeChannelGroupOpen           EventType = "group_open"
	EventTypeChannelGroupClose          EventType = "group_close"
	EventTypeChannelGroupArchive        EventType = "group_archive"
	EventTypeChannelGroupUnarchive      EventType = "group_unarchive"
	EventTypeChannelGroupRename         EventType = "group_rename"
	EventTypeChannelGroupMarked         EventType = "group_marked"
	EventTypeChannelGroupHistoryChanged EventType = "group_history_changed"
	EventTypeFileCreated                EventType = "file_created"
	EventTypeFileShared                 EventType = "file_shared"
	EventTypeFileUnshared               EventType = "file_unshared"
	EventTypeFilePublic                 EventType = "file_public"
	EventTypeFilePrivate                EventType = "file_private"
	EventTypeFileChange                 EventType = "file_change"
	EventTypeFileDeleted                EventType = "file_deleted"
	EventTypeFileCommentAdded           EventType = "file_comment_added"
	EventTypeFileCommentEdited          EventType = "file_comment_edited"
	EventTypeFileCommentDeleted         EventType = "file_comment_deleted"
	EventTypePresenceChange             EventType = "presence_change"
	EventTypeManualPresenceChange       EventType = "manual_presence_change"
	EventTypePrefChange                 EventType = "pref_change"
	EventTypeUserChange                 EventType = "user_change"
	EventTypeTeamJoin                   EventType = "team_join"
	EventTypeStarAdded                  EventType = "star_added"
	EventTypeStarRemoved                EventType = "star_removed"
	EventTypeEmojiChanged               EventType = "emoji_changed"
	EventTypeCommandsChanged            EventType = "commands_changed"
	EventTypeTeamPrefChange             EventType = "team_pref_change"
	EventTypeTeamRename                 EventType = "team_rename"
	EventTypeTeamDomainChange           EventType = "team_domain_change"
	EventTypeEmailDomainChanged         EventType = "email_domain_changed"
	EventTypeBotAdded                   EventType = "bot_added"
	EventTypeBotChanged                 EventType = "bot_changed"
	EventTypeAccountsChanged            EventType = "accounts_changed"
	EventTypeTeamMigrationStarted       EventType = "team_migration_started"
)

type Bot struct {
	ws       *websocket.Conn
	config   Config
	handlers map[EventType]handlerFunc
	id       int
	messages map[int]Message
}

type handlerFunc func(*Bot, []byte) error
type MessageFunc func(*Bot, *Message) error
type EventType string

type Timestamp time.Time

func (t *Timestamp) MarshalJSON() ([]byte, error) {
	ts := time.Time(*t).Unix()
	stamp := fmt.Sprint(ts)
	return []byte(stamp), nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var ts float64
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &ts)
	fmt.Println(err)
	fmt.Println(ts)
	if err != nil {
		return err
	}
	*t = Timestamp(time.Unix(int64(ts), 0))
	return nil
}

type Event struct {
	Id        int       `json:"id"`
	Type      string    `json:"type"`
	Channel   string    `json:"channel"`
	User      string    `json:"user"`
	Timestamp Timestamp `json:"ts"`
}

type Message struct {
	Id          int          `json:"id"`
	Type        string       `json:"type"`
	Channel     string       `json:"channel"`
	User        string       `json:"user"`
	Username    string       `json:"username"`
	BotId       string       `json:"bot_id"`
	Text        string       `json:"text"`
	Timestamp   Timestamp    `json:"ts"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Fallback    string `json:"fallback"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	ImageBytes  int    `json:"image_bytes"`
	AuthorName  string `json:"author_name"`
	Id          int    `json:"id"`
	TitleLink   string `json:"title_link"`
	FromUrl     string `json:"from_url"`
	ImageUrl    string `json:"image_url"`
	Text        string `json:"text"`
	Title       string `json:"title"`
	AuthorLink  string `json:"author_link"`
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	Channel     string `json:"channel"`
}

func MessageHandler(fn MessageFunc) handlerFunc {
	return func(b *Bot, data []byte) error {
		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			return err
		}
		err := fn(b, &message)
		return err
	}
}

func (b *Bot) SetHandler(t EventType, fn handlerFunc) {
	b.handlers[t] = fn
}

func (b *Bot) SetMessageHandler(fn MessageFunc) {
	b.SetHandler(EventTypeMessage, MessageHandler(fn))
}

func (b *Bot) reconnect() error {
	var err error

	for {
		log.Printf("Connect failed with err %s, reconnecting.\n", err)

		if err = b.connect(); err == nil {
			break
		}

		time.Sleep(15 * time.Second)

	}

	return err
}

var ErrNotSupported = errors.New("Not supported")

func (b *Bot) receive() ([]byte, error) {
	_, r, err := b.ws.NextReader()
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(r)
	if err == io.EOF {
		// Decode returns io.EOF when the message is empty or all whitespace.
		// Convert to io.ErrUnexpectedEOF so that application can distinguish
		// between an error reading the JSON value and the connection closing.
		err = io.ErrUnexpectedEOF
	}
	return data, err
}

func (b *Bot) NewMessage() Message {
	message := Message{}
	message.Id = b.id

	b.id++

	message.Type = "message"
	return message
}

func (b *Bot) Send(message Message) error {
	err := b.ws.WriteJSON(message)
	return err
}

func (b *Bot) Run() error {
	for {
		var event Event

		data, err := b.receive()

		err = json.Unmarshal(data, &event)
		if err == io.ErrUnexpectedEOF {
			err = b.reconnect()
			continue
		} else if err != nil {
			return err
		}

		fmt.Println("Received %#v", event)
		fmt.Println(string(data))

		if fn, ok := b.handlers[EventType(event.Type)]; ok {
			err := fn(b, data)
			if err != nil {
				log.Printf("Error %s\n", err)
				continue
			}
		} else {
			log.Printf("No handler found for type %s %#v.\n", event.Type, data)
		}
	}
}

type Config struct {
	PayloadToken string
	Token        string
	Origin       string
}

func (b *Bot) connect() error {
	v := url.Values{}
	v.Set("token", b.config.Token)

	resp, err := http.PostForm("https://slack.com/api/rtm.start", v)
	if err != nil {
		return err
	}

	var response struct {
		Ok    bool   `json:"ok"`
		Url   string `json:"url"`
		Error string `json:"error"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return err
	}

	if !response.Ok {
		return errors.New(response.Error)
	}

	header := http.Header{
		"Origin": {b.config.Origin},
		// your milage may differ
		"Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits, x-webkit-deflate-frame"},
	}

	b.ws, _, err = websocket.DefaultDialer.Dial(response.Url, header)
	if err != nil {
		return err
	}

	return nil
}

func NewBot(config Config) (*Bot, error) {
	bot := &Bot{config: config, id: 1, handlers: map[EventType]handlerFunc{}}

	if err := bot.connect(); err != nil {
		return nil, err
	}

	return bot, nil
}
