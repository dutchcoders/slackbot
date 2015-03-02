# slackbot
Slackbot implementation library for Go. Easy creation of new SlackCommands and Bots.

```
package main

import (
    "errors"
    "log"
    "math/rand"
    "os"
    "strings"
    "sync"
    "time"

    slackbot "github.com/dutchcoders/slackbot"
    "github.com/nlopes/slack"
)

func main() {
    engine := slackbot.NewEngine(slackbot.Config{
        PayloadToken: os.Getenv("SLACK_PAYLOAD_TOKEN"),
    })

    engine.AddCommand("/pick", pick)

    go func() {
        bot, err := slackbot.NewBot(slackbot.Config{
            Token:  os.Getenv("SLACK_TOKEN"),
            Origin: "http://localhost",
        })

        if err != nil {
            log.Println(err)
            return
        }

        bot.SetMessageHandler(func(b *slackbot.Bot, message *slackbot.Message) error {
            log.Println(message.Text)
            return nil
        })

        err = bot.Run()
        if err != nil {
            log.Println(err)
        }
    }()

    addr := ":" + os.Getenv("PORT")
    if err := engine.ListenAndServe(addr); err != nil {
        panic(err)
    }

}

func pick(sc *slackbot.Context, w http.ResponseWriter) {
    choices := strings.Split(sc.Text, ",")
    choice := choices[rand.Intn(len(choices))]
    choice = strings.Trim(choice, " ")
    fmt.Fprintf(w, "Hmmm, I'd say pick %s.", choice)
}
```

## Contributions

Contributions are welcome.

## Creators

**Remco Verhoef**
- <https://twitter.com/remco_verhoef>
- <https://twitter.com/dutchcoders>

## Copyright and license

Code and documentation copyright 2011-2014 Remco Verhoef.
Code released under [the MIT license](LICENSE).
