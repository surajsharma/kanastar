package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/surajsharma/kanastar/stats"
	"github.com/surajsharma/kanastar/task"
	"github.com/surajsharma/kanastar/utils"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	Stats     *stats.Stats
	TaskCount int
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := []*task.Task{}

	for _, t := range w.Db {
		tasks = append(tasks, t)
	}

	return tasks
}

func (w *Worker) CollectStats() {
	for {
		log.Println("[worker] collecting stats...")
		w.Stats = stats.GetStats()
		w.Stats.TaskCount = w.TaskCount
		utils.Sleep("worker", 15)

	}
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {

	t.StartTime = time.Now().UTC()

	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	result := d.Run()

	if result.Error != nil {
		log.Printf("[worker] error running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}

	t.ContainerID = result.ContainerID

	t.State = task.Running

	w.Db[t.ID] = &t

	log.Printf("[worker] started task %v\n", t.ContainerID)

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	result := d.Stop(t.ContainerID)

	if result.Error != nil {
		log.Printf("[worker] error stopping container %v: %v\n", t, result.Error)
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed

	w.Db[t.ID] = &t

	log.Printf("[worker] stopped and removed container %v for task %v\n", t.ContainerID, t.ID)

	return result
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) runTask() task.DockerResult {

	t := w.Queue.Dequeue()

	if t == nil {
		log.Println("[worker] no tasks in queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)

	taskPersisted := w.Db[taskQueued.ID]

	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = &taskQueued
	}

	var result task.DockerResult

	if task.ValidStateTransitions(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("[worker] invalid operation")
		}
	} else {
		err := fmt.Errorf("[worker] invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
		return result
	}

	return result
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("[worker] error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("[worker] no tasks to process currently\n")
		}

		utils.Sleep("worker", 10)

	}
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.Println("[worker] checking status of tasks")
		w.updateTasks()
		log.Println("[worker] tasks update completed")
		utils.Sleep("worker", 10)
	}
}

func (w *Worker) updateTasks() {
	// for each task in the worker's datastore
	// 1. call InspectTask method
	// 2. verify if task is in running state
	// 3. if task is not in running state or not running at all, set state to failed

	for id, t := range w.Db {
		if t.State == task.Running {
			resp := w.InspectTask(*t)

			if resp.Error != nil {
				fmt.Printf("[worker] error inspecting task: %v\n", resp.Error)
			}

			if resp.Container == nil {
				log.Printf("[worker] no container for running task %s\n", id)
				w.Db[id].State = task.Failed
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("[worker] container for task %s in non-running state %s\n", id, resp.Container.State.Status)
				w.Db[id].State = task.Failed
			}

			//task is running, update exposed ports
			w.Db[id].HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
		}
	}
}
