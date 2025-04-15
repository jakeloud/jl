package logger

import (
	"net/http"
	"net/url"

	"github.com/jakeloud/jl/entities"
)

func Log(message string) error {
	if message == "" {
		message = "message unspecified"
	}

	app, err := entities.GetApp(entities.JAKELOUD)
	if err != nil {
		return err
	}

	if app.Additional == nil {
		return nil
	}

	botToken, ok := app.Additional["botToken"].(string)
	if !ok {
		return nil
	}
	chatId, ok := app.Additional["chatId"].(string)
	if !ok {
		return nil
	}

	path := url.PathEscape("/bot" + botToken + "/sendMessage?chat_id=" + chatId + "&parse_mode=MarkdownV2&text=" + message)
	u := "https://api.telegram.org" + path

	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
