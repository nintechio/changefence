package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func githubEventBase() (string, error) {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	if eventName != "pull_request" && eventName != "pull_request_target" {
		return "", nil
	}
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return "", errors.New("GITHUB_EVENT_PATH is missing for pull-request event")
	}
	eventFile, err := os.Open(eventPath)
	if err != nil {
		return "", fmt.Errorf("read GitHub event: %w", err)
	}
	defer eventFile.Close()
	var event struct {
		PullRequest struct {
			Base struct {
				SHA string `json:"sha"`
			} `json:"base"`
		} `json:"pull_request"`
	}
	decoder := json.NewDecoder(io.LimitReader(eventFile, 1<<20))
	if err := decoder.Decode(&event); err != nil {
		return "", fmt.Errorf("parse GitHub event: %w", err)
	}
	if event.PullRequest.Base.SHA == "" {
		return "", errors.New("pull-request event does not contain a base commit")
	}
	return event.PullRequest.Base.SHA, nil
}
