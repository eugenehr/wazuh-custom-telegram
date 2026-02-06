package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func prepareAlert() map[string]string {
	// alert.json
	alertJson := os.Args[1]
	// Telegram chat/channel ID
	chatId := os.Args[2]

	jsonData, err := os.ReadFile(alertJson)
	if err != nil {
		panic(fmt.Sprintf("Could not read JSON data from file %s file: %s\n", alertJson, err.Error()))
	}

	jsonMap := make(map[string]any)
	err = json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		panic(fmt.Sprintf("Could not unmarshal JSON from file %s file: %s\n", alertJson, err.Error()))
	}

	var description, title, level, ruleId, groups, agentName, agentId, winSysMessage string
	if value, ok := jsonMap["full_log"]; ok {
		description = fmt.Sprintf("%v", value)
	}
	if rule, ok := jsonMap["rule"]; ok {
		ruleMap := rule.(map[string]any)
		if value, ok := ruleMap["description"]; ok {
			title = fmt.Sprintf("%v", value)
			title = strings.ReplaceAll(title, "\\n", "\n")
		}
		if value, ok := ruleMap["level"]; ok {
			level = fmt.Sprintf("%v", value)
		}
		if value, ok := ruleMap["id"]; ok {
			ruleId = fmt.Sprintf("%v", value)
		}
		if value, ok := ruleMap["groups"]; ok {
			if ruleGroups, ok := value.([]any); ok {
				for _, group := range ruleGroups {
					if len(groups) > 0 {
						groups += ", "
					}
					groups += fmt.Sprintf("%v", group)
				}
			} else {
				groups = fmt.Sprintf("%v", value)
			}
		}
	}
	if agent, ok := jsonMap["agent"]; ok {
		agentMap := agent.(map[string]any)
		if value, ok := agentMap["id"]; ok {
			agentId = fmt.Sprintf("%v", value)
		}
		if value, ok := agentMap["name"]; ok {
			agentName = fmt.Sprintf("%v", value)
		}
	}
	if data, ok := jsonMap["data"]; ok {
		if dataMap, ok := data.(map[string]any); ok {
			if win, ok := dataMap["win"]; ok {
				if winMap, ok := win.(map[string]any); ok {
					if system, ok := winMap["system"]; ok {
						if systemMap, ok := system.(map[string]any); ok {
							if message, ok := systemMap["message"]; ok {
								winSysMessage = strings.TrimSpace(fmt.Sprintf("%v", message))
								winSysMessage = strings.TrimPrefix(winSysMessage, "\"")
								winSysMessage = strings.TrimSuffix(winSysMessage, "\"")
							}
						}
					}
				}
			}
		}
	}

	message := "<b>" + html.EscapeString(title) + "</b>\n\n"
	message += html.EscapeString(description) + "\n"
	if len(winSysMessage) > 0 {
		message += "\n" + html.EscapeString(winSysMessage) + "\n"
	}
	message += "\n"
	message += "<b>Agent: </b>" + html.EscapeString(agentName) + " (Id " + html.EscapeString(agentId) + ")\n"
	message += "<b>Rule: </b>" + html.EscapeString(ruleId) + " (Level " + html.EscapeString(level) + ")\n"
	message += "<b>Groups: </b>" + html.EscapeString(groups) + "\n"

	message = strings.ReplaceAll(message, "\r", "")
	message = regexp.MustCompile("\n{3,}").ReplaceAllString(message, "\n\n")

	msg := make(map[string]string)
	msg["chat_id"] = chatId
	msg["text"] = message
	msg["parse_mode"] = "HTML"

	return msg
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: custom-telegram <alert.json> <chat_id> <hook_url>")
		os.Exit(1)
	}

	// Telegram sendMessage full url
	hookUrl := os.Args[3]

	msgData, err := json.Marshal(prepareAlert())
	if err != nil {
		panic(err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, hookUrl, bytes.NewBuffer(msgData))
	req.Header.Set("Content-Type", "application/json")

	_, _ = http.DefaultClient.Do(req)
}
