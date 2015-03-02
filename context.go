package slackbot

type Context struct {
	Token       string
	TeamID      string
	ChannelID   string
	ChannelName string
	UserName    string
	UserID      string
	Command     string
	TriggerWord string
	Text        string
}
