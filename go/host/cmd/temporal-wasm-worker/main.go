package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cretz/temporal-wasm/go/host"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// TODO(cretz): Make this a full cobra app w/ flags and all
	if len(os.Args) != 5 {
		return fmt.Errorf("expecting 4 args: server addr, task queue, workflow name, workflow file")
	}

	log.Printf("Connecting to server")
	c, err := client.NewClient(client.Options{HostPort: os.Args[1]})
	if err != nil {
		return fmt.Errorf("failed creating client: %w", err)
	}
	defer c.Close()

	log.Printf("Loading WASM")
	workflowFn, err := host.NewWASMWorkflow(host.WASMFromFile(os.Args[4]))
	if err != nil {
		return fmt.Errorf("failed creating workflow: %w", err)
	}

	log.Printf("Starting worker")
	w := worker.New(c, os.Args[2], worker.Options{})
	w.RegisterWorkflowWithOptions(workflowFn, workflow.RegisterOptions{Name: os.Args[3]})
	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("worker failed: %w", err)
	}
	return nil
}
