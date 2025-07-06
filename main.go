package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/moritz-tiesler/sous/agent"
	"github.com/ollama/ollama/api"
)

func main() {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	agentInstance := agent.NewAgent(client, getUserMessage)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	appCtx, appCancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-sigCh:
				if agentInstance.IsRunning() {
					agentInstance.Cancel()
				} else {
					appCancel()
				}
			case <-appCtx.Done():
				return
			}
		}
	}()

	go func() {
		err = agentInstance.Run(appCtx)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			appCancel()
		}
	}()

	<-appCtx.Done()
	fmt.Println("Bye")
	os.Exit(1)
}