package cmd

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/surajsharma/kanastar/utils"
	"github.com/surajsharma/kanastar/worker"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker command to operate a Kanastar worker node.",
	Long: `Kanastar worker command.

	The worker runs tasks and responds to the manager's requests about task state.`,

	Run: func(cmd *cobra.Command, args []string) {

		if !utils.IsDockerDaemonUp() {
			return
		}

		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		name, _ := cmd.Flags().GetString("name")
		dbType, _ := cmd.Flags().GetString("dbtype")

		w := worker.New(name, dbType)
		log.Printf("[cmd] starting worker %s", w.Name)
		api := worker.Api{Address: host, Port: port, Worker: w}
		go w.RunTasks()
		go w.CollectStats()
		go w.UpdateTasks()
		log.Printf("[cmd] starting worker API on http://%s:%d", host, port)
		api.Start()
	}}

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.Flags().StringP("host", "H", "0.0.0.0", "Hostname or IP address")
	workerCmd.Flags().IntP("port", "p", 5556, "Port on which to listen")
	workerCmd.Flags().StringP("name", "n", fmt.Sprintf("worker-%s", uuid.New().String()), "Name of the worker")
	workerCmd.Flags().StringP("dbtype", "d", "memory", "Type of datastore to use for tasks (\"memory\" or \"persistent\")")

}
