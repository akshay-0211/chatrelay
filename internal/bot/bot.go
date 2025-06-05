package bot

import (
	"bytes"
	"chatrelay/internal/config"
	telemetry "chatrelay/internal/telementry"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"go.opentelemetry.io/otel/attribute"
)

func StartSlackBot() error {
	appToken := config.GetEnv("SLACK_APP_TOKEN")
	botToken := config.GetEnv("SLACK_BOT_TOKEN")

	api := slack.New(
		botToken,
		slack.OptionDebug(true),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(api)

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}
				client.Ack(*evt.Request)

				switch ev := eventsAPIEvent.InnerEvent.Data.(type) {
				case *slackevents.AppMentionEvent:
					handleMessage(api, ev.Channel, ev.User, ev.Text)
				case *slackevents.MessageEvent:
					if ev.ChannelType == "im" && ev.BotID == "" {
						handleMessage(api, ev.Channel, ev.User, ev.Text)
					}
				}
			}
		}
	}()

	return client.Run()
}

func handleMessage(api *slack.Client, channel, user, text string) {
	_, span := telemetry.Tracer.Start(context.Background(), "HandleSlackMessage")
	span.SetAttributes(
		attribute.String("user.id", user),
		attribute.String("channel.id", channel),
		attribute.String("message.text", text),
	)
	defer span.End()

	api.PostMessage(channel, slack.MsgOptionText("Processing your query...", false))

	payload := map[string]string{
		"user_id": user,
		"query":   text,
	}

	var resp *http.Response
	var err error
	const maxRetries = 3
	var backendStart = time.Now()

	for i := 0; i < maxRetries; i++ {
		body, _ := json.Marshal(payload)
		resp, err = http.Post(config.GetEnv("BACKEND_URL"), "application/json", bytes.NewBuffer(body))
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond) // backoff
	}

	backendDuration := time.Since(backendStart).Milliseconds()
	span.SetAttributes(attribute.Float64("backend.duration.ms", float64(backendDuration)))

	if err != nil || resp == nil {
		api.PostMessage(channel, slack.MsgOptionText("Backend unreachable after retries.", false))
		span.RecordError(err)
		return
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result["full_response"] == "" {
		api.PostMessage(channel, slack.MsgOptionText("Invalid backend response.", false))
		span.RecordError(err)
		return
	}

	// Stream simulated chunks
	chunks := strings.Split(result["full_response"], ". ")
	for _, chunk := range chunks {
		if chunk != "" {
			api.PostMessage(channel, slack.MsgOptionText(chunk+".", false))
			time.Sleep(500 * time.Millisecond)
		}
	}
}
