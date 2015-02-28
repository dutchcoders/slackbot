package slackbot

type Command struct {
	Token       string
	TeamID      string
	ChannelID   string
	ChannelName string
	UserName    string
	UserID      string
	Command     string
	Text        string
}

/*
func SlackCommandHandler(command string, fn func(sc *SlackCommand, w http.ResponseWriter)) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		sc := &SlackCommand{}
		sc.token = req.FormValue("token")
		sc.team_id = req.FormValue("team_id")
		sc.channel_id = req.FormValue("channel_id")
		sc.channel_name = req.FormValue("channel_name")
		sc.user_name = req.FormValue("user_name")
		sc.user_id = req.FormValue("user_id")
		sc.command = req.FormValue("command")
		sc.text = req.FormValue("text")

		if sc.command != command {
			return
		}

		fn(sc, w)
	}
}
*/
