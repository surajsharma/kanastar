package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/surajsharma/kanastar/task"
	"github.com/surajsharma/kanastar/worker"
)

type Manager struct {
	Pending       queue.Queue
	TaskDb        map[uuid.UUID]*task.Task
	EventDb       map[uuid.UUID]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
}

func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing any tasks in the queue")
		m.SendWork()
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) UpdateTasks() {
	for {
		log.Println("Checking for task updates from worker")
		m.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func New(workers []string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)

	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
	}
}

func (m *Manager) SelectWorker() string {
	var newWorker int

	if m.LastWorker+1 < len(m.Workers) {
		newWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		newWorker = 0
		m.LastWorker = 0
	}

	return m.Workers[newWorker]
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()

		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)

		t := te.Task
		log.Printf("Pulled %v off pending queue\n", t)

		m.EventDb[te.ID] = &te
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)
		m.TaskWorkerMap[t.ID] = w

		t.State = task.Scheduled
		m.TaskDb[t.ID] = &t

		data, err := json.Marshal(te)

		if err != nil {
			log.Printf("Unable to marshal task object: %v\n", t)
		}

		url := fmt.Sprintf("http://%s/tasks", w)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

		if err != nil {
			log.Printf("Error connecting to %v: %v\n", w, err)
			m.Pending.Enqueue(te)
			return
		}

		d := json.NewDecoder(resp.Body)

		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}

			err := d.Decode(&e)

			if err != nil {
				fmt.Printf("Error decoding response: %s\n", err.Error())
				return
			}

			log.Printf("Response error (%d): %s\n", e.HTTPStatusCode, e.Message)
			return
		}

		log.Printf("%#v\n", t)

	} else {
		log.Println("No work in the queue")
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}

	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}

	return tasks

}

func (m *Manager) DoHealthChecks() {
	for {
		log.Printf("Performing task health check")
		m.doHealthChecks()
		log.Printf("Task health checks completed")
		log.Printf("Sleeping for 60 seconds")
		time.Sleep(60 * time.Second)

	}
}

func (m *Manager) updateTasks() {
	for _, worker := range m.Workers {
		log.Printf("Checking worker %v for task updates\n", worker)

		url := fmt.Sprintf("http://%s/tasks", worker)

		resp, err := http.Get(url)

		if err != nil {
			log.Printf("Error connecting to %v:%v, retrying in 5 seconds", worker, err)
			time.Sleep(5 * time.Second)
			defer m.updateTasks()
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error sending request: %v\n", err)
		}

		d := json.NewDecoder(resp.Body)

		var tasks []*task.Task

		err = d.Decode(&tasks)

		if err != nil {
			log.Printf("Error unmarshalling tasks: %s\n", err.Error())
		}

		for _, t := range tasks {
			log.Printf("Attempting to update task %v\n", t.ID)

			_, ok := m.TaskDb[t.ID]

			if !ok {
				log.Printf("Task with ID %s not found", t.ID)
				return
			}

			if m.TaskDb[t.ID].State != t.State {
				m.TaskDb[t.ID].State = t.State
			}

			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}

	}
}

func (m *Manager) restartTask(t *task.Task) {
	//get the worker where task was runnng

	w := m.TaskWorkerMap[t.ID]

	t.State = task.Scheduled

	t.RestartCount++

	//we need to overwrite the existing task to ensure it has current state
	m.TaskDb[t.ID] = t

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}

	data, err := json.Marshal(te)

	if err != nil {
		log.Printf("Unable to marshal task object: %v\n", t)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

	if err != nil {
		log.Printf("Error connecting to %v: %v\n", w, err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		e := worker.ErrResponse{}

		err := d.Decode(&e)

		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}

		log.Printf("Response err (%d): %s\n", e.HTTPStatusCode, e.Message)
		return
	}

	newTask := task.Task{}

	err = d.Decode(&newTask)

	if err != nil {
		fmt.Printf("Error decoding response: %s\n", err.Error())
		return
	}

	log.Printf("%#v\n", t)

}

func (m *Manager) doHealthChecks() {
	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < 3 {
			err := m.checkHealthTask(*t)

			if err != nil {
				if t.RestartCount < 3 {
					m.restartTask(t)
				}
			} else if t.State == task.Failed && t.RestartCount < 3 {
				m.restartTask(t)
			}
		}
	}
}

func (m *Manager) checkHealthTask(t task.Task) error {
	log.Printf("Calling health check for task %s: %s\n", t.ID, t.HealthCheck)

	w := m.TaskWorkerMap[t.ID]

	worker := strings.Split(w, ":")

	hostPort := getHostPort(t.HostPorts)
	if hostPort == nil {
		log.Printf("Have not collected task %s host port yet. Skipping.\n", t.ID)
		return nil
	}

	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostPort, t.HealthCheck)
	log.Printf("Calling health check for task %s: %s\n", t.ID, url)

	resp, err := http.Get(url)

	if err != nil {
		msg := fmt.Sprintf("Error connecting to health check url %s", url)
		log.Println(msg)
		return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Health check for %s failed with status %v\n", t.ID, resp.StatusCode)
		log.Println(msg)
		return errors.New(msg)
	}

	log.Printf("Task %s health check response: %v\n", t.ID, resp.StatusCode)
	return nil

}

func getHostPort(ports nat.PortMap) *string {
	for k := range ports {
		if len(ports[k]) > 0 {
			hostPort := ports[k][0].HostPort
			return &hostPort
		}
	}
	return nil
}

// func handlePanic() {
// 	if r := recover(); r != nil {
// 		log.Printf("Recovered from panic: %v", r)
// 	}
// }
