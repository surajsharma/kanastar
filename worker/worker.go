package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/surajsharma/kanastar/stats"
	"github.com/surajsharma/kanastar/store"
	"github.com/surajsharma/kanastar/task"
	"github.com/surajsharma/kanastar/utils"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        store.Store
	Stats     *stats.Stats
	TaskCount int
}

func New(name string, taskDbType string) *Worker {
	w := Worker{
		Name:  name,
		Queue: *queue.New(),
	}

	var s store.Store
	var err error

	switch taskDbType {
	case "memory":
		s = store.NewInMemoryTaskStore()
	case "persistent":
		filename := fmt.Sprintf("%s_tasks.db", name)
		s, err = store.NewTaskStore(filename, 0600, "tasks")
	}

	if err != nil {
		log.Printf("eunable to create new task store: %v", err)
	}

	w.Db = s
	return &w
}

func (w *Worker) GetTasks() []*task.Task {
	taskList, err := w.Db.List()
	if err != nil {
		log.Printf("error getting list of tasks: %v\n", err)
		return nil
	}

	return taskList.([]*task.Task)
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
		w.Db.Put(t.ID.String(), &t)
		return result
	}

	t.ContainerID = result.ContainerID

	t.State = task.Running

	w.Db.Put(t.ID.String(), &t)

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

	w.Db.Put(t.ID.String(), &t)

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
	fmt.Printf("[worker] Found task in queue: %v:\n", taskQueued)

	err := w.Db.Put(taskQueued.ID.String(), &taskQueued)
	if err != nil {
		msg := fmt.Errorf("error storing task %s: %v", taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	result, err := w.Db.Get(taskQueued.ID.String())
	if err != nil {
		msg := fmt.Errorf("error getting task %s from database: %v", taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	taskPersisted := *result.(*task.Task)

	if taskPersisted.State == task.Completed {
		return w.StopTask(taskPersisted)
	}

	var dockerResult task.DockerResult

	if task.ValidStateTransitions(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			dockerResult = w.StartTask(taskQueued)
		case task.Completed:
			dockerResult = w.StopTask(taskQueued)
		default:
			dockerResult.Error = errors.New("[worker] invalid operation")
		}
	} else {
		err := fmt.Errorf("[worker] invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		dockerResult.Error = err
		return dockerResult
	}

	return dockerResult
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

	// for each task in the worker's datastore:
	// 1. call InspectTask method
	// 2. verify task is in running state
	// 3. if task is not in running state, or not running at all, mark task as `failed`
	tasks, err := w.Db.List()
	if err != nil {
		log.Printf("[worker] error getting list of tasks: %v\n", err)
		return
	}
	for _, t := range tasks.([]*task.Task) {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				fmt.Printf("[worker] error inspecting task %v\n", resp.Error)
			}

			if resp.Container == nil {
				log.Printf("[worker] no container for running task %s\n", t.ID)
				t.State = task.Failed
				w.Db.Put(t.ID.String(), t)
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("[worker] container for task %s in non-running state %s\n", t.ID, resp.Container.State.Status)
				t.State = task.Failed
				w.Db.Put(t.ID.String(), t)
			}

			// task is running, update exposed ports
			t.HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
			w.Db.Put(t.ID.String(), t)
		}
	}
}
