package worker

import (
	"encoding/json"
	"fmt"
	"github.com/surajsharma/orcs/task"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)

	d.DisallowUnknownFields()

	te := task.TaskEvent{}

	err := d.Decode(&te)

	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling request body: %v\n", err)
		log.Printf("%s", msg)
		w.WriteHeader(http.StatusBadRequest)

		e := ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        msg,
		}

		json.NewEncoder(w).Encode(e)
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task %v", te.Task.ID)
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	if taskID == "" {
		log.Printf("No taskID passed in request.\n")
		w.WriteHeader(http.StatusBadRequest)
	}

	tID, _ := uuid.Parse(taskID)

	_, ok := a.Worker.Db[tID]

	if !ok {
		defer handlePanic()
		w.WriteHeader(http.StatusNotFound)

		err := fmt.Sprintf("Task not found %v", tID)
		w.Write([]byte(err))
	}

	taskToStop := a.Worker.Db[tID]

	// make a copy to not modify the task in datastore
	taskCopy := *taskToStop

	taskCopy.State = task.Completed

	a.Worker.AddTask(taskCopy)

	log.Printf("Added task %v to stop container %v\n", taskToStop.ID, taskToStop.ContainerID)
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func handlePanic() {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic: %v", r)
	}
}
