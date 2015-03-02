package slackbot

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"errors"
	"io"
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

type handlerFunc func(*Bot, map[string]interface{}) error
type MessageFunc func(*Bot, *Message) error
type EventType string

type Message struct {
	Id      int    `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	User    string `json:"user"`
	Text    string `json:"text"`
}

type Event struct {
	Type string
}

func MessageHandler(fn MessageFunc) handlerFunc {
	return func(b *Bot, data map[string]interface{}) error {
		var message Message
		if err := merge(&message, data); err != nil {
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

func merge(dst interface{}, src map[string]interface{}) error {
	vDst := reflect.ValueOf(dst).Elem()
	if vDst.Kind() != reflect.Struct {
		return ErrNotSupported
	}

	vSrc := reflect.ValueOf(src)
	if vSrc.Kind() != reflect.Map {
		return ErrNotSupported
	}

	return deepMerge(dst, src)
}

func deepMerge(dst interface{}, src map[string]interface{}) error {
	tDst := reflect.TypeOf(dst).Elem()
	for i := 0; i < tDst.NumField(); i++ {
		tField := tDst.Field(i)
		fieldName := strings.ToLower(tField.Name)
		if _, ok := src[fieldName]; !ok {
			continue
		}
		// if field is struct, and src is map, recurse
		val := reflect.ValueOf(src[fieldName])
		reflect.ValueOf(dst).Elem().Field(i).Set(val)
	}
	return nil
}

func (b *Bot) receive(data *map[string]interface{}) error {
	//err := websocket.JSON.Receive(b.ws, &data)
	err := b.ws.ReadJSON(&data)
	return err
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
		var data map[string]interface{}

		err := b.receive(&data)
		if err == io.EOF {
			err = b.reconnect()
			continue
		} else if err != nil {
			return err
		}

		var event Event
		if err = merge(&event, data); err != nil {
			log.Printf("Error merging %#v.\n", err)
			continue
		}

		fmt.Println("Received %#v", data)
		/*
			var confirmation struct {
				Ok        bool   `json:"ok"`
				ReplyTo   int    `json:"reply_to"`
				TimeStamp string `json:"ts"`
				Text      string `json:"text"`
			}

			// wait for confirmation
			if err := websocket.JSON.Receive(b.ws, &confirmation); err != nil {
				return err
			}

			if confirmation.ReplyTo != message.Id {
			}

		*/
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
