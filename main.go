package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

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

	fmt.Println("Starting Orcs Worker")

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	wapi := worker.Api{Address: whost, Port: wport, Worker: &w}

	go w.CollectStats()
	go runTasks(&w)

	go wapi.Start()

	log.Println("Waiting for API to start...")
	time.Sleep(10 * time.Second)

	fmt.Println("Starting orcs manager")
	workers := []string{fmt.Sprintf("%s: %d", whost, wport)}
	m := manager.New(workers)
	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasksForever()
	mapi.Start()

	for i := 0; i < 3; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}

		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}

		m.AddTask(te)
		m.SendWork()
	}

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
