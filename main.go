package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/surajsharma/kanastar/manager"
	"github.com/surajsharma/kanastar/worker"
)

func main() {

	mhost := os.Getenv("MANAGER_HOST")
	mport, _ := strconv.Atoi(os.Getenv("MANAGER_PORT"))

	whost := os.Getenv("WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("WORKER_PORT"))

	fmt.Println("⏳ Starting worker...")

	w1 := worker.New("w1", "persistent")
	w2 := worker.New("w2", "persistent")
	w3 := worker.New("w3", "persistent")

	wapi1 := worker.Api{Address: whost, Port: wport, Worker: w1}
	wapi2 := worker.Api{Address: whost, Port: wport + 1, Worker: w2}
	wapi3 := worker.Api{Address: whost, Port: wport + 2, Worker: w3}

	go w1.RunTasks()
	go w1.CollectStats()
	go w1.UpdateTasks()
	go wapi1.Start()

	go w2.RunTasks()
	go w2.CollectStats()
	go w2.UpdateTasks()
	go wapi2.Start()

	go w3.RunTasks()
	go w3.CollectStats()
	go w3.UpdateTasks()
	go wapi3.Start()

	fmt.Println("⏳ Starting manager...")

	workers := []string{
		fmt.Sprintf("%s:%d", whost, wport),
		fmt.Sprintf("%s:%d", whost, wport+1),
		fmt.Sprintf("%s:%d", whost, wport+2),
	}

	m := manager.New(workers, "epvm", "memory")

	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.Start() //this cannot be a goroutine for http.ListenAndServe is blocking
}
