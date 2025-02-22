package utils

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/client"
)

func IsDockerDaemonUp() bool {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("[docker] error creating docker client, is the daemon running?")
		return false
	}

	_, pingErr := cli.Ping(context.Background())
	if pingErr != nil {
		fmt.Printf("[docker] daemon is not running, please start docker before using Kanastar\n")
		return false
	} else {
		fmt.Printf("[docker] daemon is running, using client version %v\n", cli.ClientVersion())
		return true
	}
}
