package slackbot

import (
	"log"
	"net/http"
)

type SlackFunc func(sc *Command, w http.ResponseWriter)

func NewEngine(token string) *Engine {
	commands := make(map[string]SlackFunc)
	return &Engine{commands: commands, token: token}
}

type Engine struct {
	commands map[string]SlackFunc
	token    string
}

func (e *Engine) ListenAndServe(addr string) error {
	err := http.ListenAndServe(addr, e)
	return err
}

func (engine *Engine) Add(cmd string, fn SlackFunc) {
	engine.commands[cmd] = fn
}

// ServeHTTP makes the router implement the http.Handler interface.
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sc := &Command{}
	sc.Token = req.PostFormValue("token")
	sc.TeamID = req.PostFormValue("team_id")
	sc.ChannelID = req.PostFormValue("channel_id")
	sc.ChannelName = req.PostFormValue("channel_name")
	sc.UserName = req.PostFormValue("user_name")
	sc.UserID = req.PostFormValue("user_id")
	sc.Command = req.PostFormValue("command")
	sc.Text = req.PostFormValue("text")

	// test token for valid token
	if sc.Token != e.token {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		log.Printf("403. Invalid token: %s", sc.Token)
		return
	}

	if fn, ok := e.commands[sc.Command]; ok {
		log.Printf("Command %s %s %s %s", sc.UserName, sc.ChannelName, sc.Command, sc.Text)
		fn(sc, w)
	} else {
		http.NotFound(w, req)
	}
}
