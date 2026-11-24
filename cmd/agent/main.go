package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"realchat/internal/llm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := llm.ConfigFromEnv()
	if err != nil {
		return err
	}
	client := llm.NewClient(cfg)

	history := []llm.Message{
		{Role: llm.RoleSystem, Content: "You are a concise assistant."},
	}

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")
		if !in.Scan() {
			return in.Err()
		}
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		if line == "exit" {
			return nil
		}

		history = append(history, llm.Message{Role: llm.RoleUser, Content: line})
		reply, err := client.Chat(context.Background(), history, nil)
		if err != nil {
			return err
		}
		history = append(history, reply)
		fmt.Println(reply.Content)
	}
}
