package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/surajsharma/orcs/task"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}

	err := d.Decode(&te)

	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Print(msg)
		w.WriteHeader(http.StatusBadRequest)

		e := ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        msg,
		}

		json.NewEncoder(w).Encode(e)
		return
	}

	a.Manager.AddTask(te)
	log.Printf("Added task: %v\n", te.Task.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)

}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	if taskID == "" {
		log.Printf("No TaskID passed in request\n")
		w.WriteHeader(http.StatusBadRequest)
	}
	tID, _ := uuid.Parse(taskID)

	taskToStop, ok := a.Manager.TaskDb[tID]

	if !ok {
		log.Printf("Task ID %v not found", tID)
		w.WriteHeader(http.StatusNotFound)
	}

	te := task.TaskEvent{

		ID:        uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now(),
	}

	taskCopy := *taskToStop

	taskCopy.State = task.Completed

	te.Task = taskCopy

	a.Manager.AddTask(te)

	log.Printf("Added task event %v to stop task %v \n", te.ID, taskToStop.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Manager.GetTasks())
}
