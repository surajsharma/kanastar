package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/surajsharma/orcs/task"
	"github.com/surajsharma/orcs/worker"
)

func main() {

	host := os.Getenv("ORCS_HOST")
	port, _ := strconv.Atoi(os.Getenv("ORCS_PORT"))

	fmt.Println("Starting Orcs Worker")

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.Api{Address: host, Port: port, Worker: &w}

	go w.CollectStats()

	go runTasks(&w)

	api.Start()
}

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("No tasks to process currently.\n")
		}

		log.Printf("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)

	}
}
