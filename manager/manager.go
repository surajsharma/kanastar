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
	"github.com/surajsharma/kanastar/node"
	"github.com/surajsharma/kanastar/scheduler"
	"github.com/surajsharma/kanastar/task"
	"github.com/surajsharma/kanastar/utils"
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
	WorkerNodes   []*node.Node
	Scheduler     scheduler.Scheduler
}

func New(workers []string, schedulerType string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)

	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	var nodes []*node.Node

	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}

		nAPI := fmt.Sprintf("http://%v", workers[worker])
		n := node.NewNode(workers[worker], nAPI, "worker")
		nodes = append(nodes, n)
	}

	var s scheduler.Scheduler

	switch schedulerType {
	case "greedy":
		s = &scheduler.RoundRobin{Name: "greedy"}
	case "roundrobin":
		s = &scheduler.RoundRobin{Name: "roundrobin"}
	default:
		s = &scheduler.RoundRobin{Name: "epvm"}
	}

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
		WorkerNodes:   nodes,
		Scheduler:     s,
	}
}

func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)

	if candidates == nil {
		msg := fmt.Sprintf("[manager] no available candidates match for task %v\n", t.ID)
		err := errors.New(msg)
		return nil, err
	}

	scores := m.Scheduler.Score(t, candidates)

	if scores == nil {
		return nil, fmt.Errorf("[manager] no scores returned to task %v", t)
	}

	selectedNode := m.Scheduler.Pick(scores, candidates)
	return selectedNode, nil
}

func (m *Manager) ProcessTasks() {
	for {
		log.Println("[manager] processing any tasks in the queue")
		m.SendWork()
		utils.Sleep("manager", 10)
	}
}

func (m *Manager) UpdateTasks() {
	for {
		log.Println("[manager] checking for task updates from worker")
		m.updateTasks()
		log.Println("[manager] task updates completed")
		utils.Sleep("manager", 10)
	}
}

func (m *Manager) updateTasks() {
	for _, worker := range m.Workers {

		log.Printf("[manager] checking worker %v for task updates\n", worker)

		url := fmt.Sprintf("http://%s/tasks", worker)

		resp, err := http.Get(url)

		if err != nil {
			log.Printf("[manager] error connecting to %v:%v, retrying in 5 seconds", worker, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[manager] error sending request to worker url: %v\n", err)
			continue
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)

		if err != nil {
			log.Printf("[manager] error unmarshalling tasks: %s\n", err.Error())
		}

		for _, t := range tasks {
			log.Printf("[manager] attempting to update task %v\n", t.ID)

			_, ok := m.TaskDb[t.ID]

			if !ok {
				log.Printf("[manager] task with ID %s not found", t.ID)
				return
			}

			if m.TaskDb[t.ID].State != t.State {
				m.TaskDb[t.ID].State = t.State
			}

			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
			m.TaskDb[t.ID].HostPorts = t.HostPorts
		}

	}
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {

		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)

		m.EventDb[te.ID] = &te
		log.Printf("[manager] pulled %v off pending queue\n", te)

		t := te.Task
		w, err := m.SelectWorker(t)
		if err != nil {
			log.Printf("[manager] error selecting worker for task %s : %v", t.ID, err)
			return
		}

		log.Printf("[manager] selected worker [%s] for task [%s]", w.Name, t.ID)

		_, ok := m.TaskWorkerMap[te.Task.ID]

		if ok {
			persistedTask := m.TaskDb[te.Task.ID]
			if te.State == task.Completed {
				m.stopTask(w, te.Task.ID.String())
				return
			}

			if !task.ValidStateTransitions(persistedTask.State, te.State) {
				log.Printf("[manager] invalid request: existing task %s is in state %v and cannot transition to the completed state", persistedTask.ID.String(), persistedTask.State)
				return
			}
		}

		m.WorkerTaskMap[w.Name] = append(m.WorkerTaskMap[w.Name], te.Task.ID)
		m.TaskWorkerMap[t.ID] = w.Name

		t.State = task.Scheduled
		m.TaskDb[t.ID] = &t

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf("[manager] unable to marshal task object: %v\n", t)
		}

		url := fmt.Sprintf("%s/tasks", w.Api)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("[manager] error connecting to %v: %v\n", w, err)
			m.Pending.Enqueue(te)
			return
		}

		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			err := d.Decode(&e)
			if err != nil {
				fmt.Printf("[manager] error decoding response: %s\n", err.Error())
				return
			}
			log.Printf("[manager] response error (%d): %s\n", e.HTTPStatusCode, e.Message)
			return
		}

		t = task.Task{}
		err = d.Decode(&t)
		if err != nil {
			fmt.Printf("[manager] error decoding response: %s\n", err.Error())
			return
		}

		w.TaskCount++
		log.Printf("[manager] received response from worker: %#v\n", t)

	} else {
		log.Println("[manager] no work in the queue")
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
		log.Printf("[manager] performing task health check..")
		m.doHealthChecks()
		log.Printf("[manager] task health checks completed")
		utils.Sleep("manager", 60)
	}
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
		log.Printf("[manager] error connecting to %v: %v\n", w, err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		e := worker.ErrResponse{}

		err := d.Decode(&e)

		if err != nil {
			fmt.Printf("[manager] error decoding response: %s\n", err.Error())
			return
		}

		log.Printf("[manager] response err (%d): %s\n", e.HTTPStatusCode, e.Message)
		return
	}

	newTask := task.Task{}

	err = d.Decode(&newTask)

	if err != nil {
		fmt.Printf("[manager] error decoding response: %s\n", err.Error())
		return
	}

	log.Printf("%#v\n", t)

}

func (m *Manager) stopTask(worker *node.Node, taskID string) {

	client := &http.Client{}

	url := fmt.Sprintf("%s/tasks/%s", worker.Api, taskID)

	req, err := http.NewRequest("DELETE", url, nil)

	if err != nil {
		log.Printf("[manager] error creating request to delete task %s: %v", taskID, err)
		return
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("[manager] error connecting to worker at %s : %v", url, err)
		return
	}

	if resp.StatusCode != http.StatusNoContent {
		log.Printf("[manager] rror connecting sending request: %v", err)
		return
	}

	log.Printf("[manager] task %s has been scheduled to be stopped", taskID)

}

func (m *Manager) checkHealthTask(t task.Task) error {
	log.Printf("[manager] calling health check for task %s: %s\n", t.ID, t.HealthCheck)

	w := m.TaskWorkerMap[t.ID]

	worker := strings.Split(w, ":")

	hostPort := getHostPort(t.HostPorts)

	if hostPort == nil {
		log.Printf("[manager] have not collected task %s host port yet. Skipping.\n", t.ID)
		return nil
	}

	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostPort, t.HealthCheck)
	log.Printf("[manager] calling health check for task %s: %s\n", t.ID, url)

	resp, err := http.Get(url)

	if err != nil {
		msg := fmt.Sprintf("[manager] error connecting to health check url %s", url)
		log.Println(msg)
		return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Health check for %s failed with status %v\n", t.ID, resp.StatusCode)
		log.Println(msg)
		return errors.New(msg)
	}

	log.Printf("[manager] task %s health check response: %v\n", t.ID, resp.StatusCode)
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
