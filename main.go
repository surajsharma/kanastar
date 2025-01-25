package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/surajsharma/orcs/manager"
	"github.com/surajsharma/orcs/task"
	"github.com/surajsharma/orcs/worker"
)

func main() {

	mhost := os.Getenv("MANAGER_HOST")
	mport, _ := strconv.Atoi(os.Getenv("MANAGER_PORT"))

	whost := os.Getenv("WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("WORKER_PORT"))

	fmt.Println("Starting Cube worker")

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	wapi := worker.Api{Address: whost, Port: wport, Worker: &w}

	go w.RunTasks()
	go w.CollectStats()
	go wapi.Start()

	fmt.Println("Starting Cube manager")

	workers := []string{fmt.Sprintf("%s:%d", whost, wport)}
	m := manager.New(workers)
	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()

	mapi.Start()

}
