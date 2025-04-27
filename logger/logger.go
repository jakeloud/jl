package logger

import (
	"fmt"
	_ "io"
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

	url := fmt.Sprintf(
		"https://api.telegram.org/bot%s/sendMessage?chat_id=%s&parse_mode=MarkdownV2&text=%s",
		botToken,
		chatId,
		url.QueryEscape(message),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	/*
	  body, err := io.ReadAll(resp.Body)
	  if err != nil {
	    return err
	  }
	  fmt.Printf("%s\n%s\n", url, body)
	*/

	return nil
}
