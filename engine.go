package slackbot

import (
	"log"
	"net/http"
)

type SlackFunc func(sc *Context, w http.ResponseWriter)

func NewEngine(config Config) *Engine {
	commands := make(map[string]SlackFunc)
	return &Engine{commands: commands, token: config.PayloadToken}
}

type Engine struct {
	commands map[string]SlackFunc
	hooks    map[string]SlackFunc
	token    string
}

func (e *Engine) ListenAndServe(addr string) error {
	err := http.ListenAndServe(addr, e)
	return err
}

func (engine *Engine) AddCommand(cmd string, fn SlackFunc) {
	engine.commands[cmd] = fn
}

func (engine *Engine) AddHook(trigger string, fn SlackFunc) {
	engine.hooks[trigger] = fn
}

// ServeHTTP makes the router implement the http.Handler interface.
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sc := &Context{}
	sc.Token = req.PostFormValue("token")
	sc.TeamID = req.PostFormValue("team_id")
	sc.ChannelID = req.PostFormValue("channel_id")
	sc.ChannelName = req.PostFormValue("channel_name")
	sc.UserName = req.PostFormValue("user_name")
	sc.UserID = req.PostFormValue("user_id")
	sc.Command = req.PostFormValue("command")
	sc.TriggerWord = req.PostFormValue("trigger_word")
	sc.Text = req.PostFormValue("text")

	// test token for valid token
	if sc.Token != e.token {
		//	http.Error(w, "Invalid token", http.StatusUnauthorized)
		//	log.Printf("403. Invalid token: %s", sc.Token)
		//	return
	}

	if fn, ok := e.commands[sc.Command]; ok {
		log.Printf("Command %s %s %s %s", sc.UserName, sc.ChannelName, sc.Command, sc.Text)
		fn(sc, w)
	} else if fn, ok := e.hooks[sc.TriggerWord]; ok {
		log.Printf("Webhook %s %s %s %s", sc.UserName, sc.ChannelName, sc.TriggerWord, sc.Text)
		fn(sc, w)
	} else if fn, ok := e.hooks[sc.ChannelName]; ok {
		log.Printf("Webhook %s %s %s %s", sc.UserName, sc.ChannelName, sc.TriggerWord, sc.Text)
		fn(sc, w)
	} else {
		http.NotFound(w, req)
	}
}
