package main

import (
	"fmt"
	"github.com/pborman/uuid"
	"github.com/re-V01t/Cadence/src/common"
	"github.com/re-V01t/Cadence/src/workflows"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
	"time"
)

func main() {
	fmt.Println("STARTED!!!!!")
	var h = common.Helper{}
	const ApplicationName string = "halfblood"
	h.Setup(&common.Configuration{
		DomainName:      "cadence-test",
		ServiceName:     "cadence-frontend",
		HostNameAndPort: "127.0.0.1:7833",
		//Prometheus:      nil,
	})
	h.RegisterWorkflowWithAlias(workflows.SimpleWorkflow, "hello_world")
	h.RegisterActivity(workflows.SimpleActivity)
	// Configure worker options.
	workerOptions := worker.Options{
		MetricsScope: h.WorkerMetricScope,
		Logger:       h.Logger,
		FeatureFlags: client.FeatureFlags{
			WorkflowExecutionAlreadyCompletedErrorEnabled: true,
		},
	}
	h.StartWorkers(h.Config.DomainName, ApplicationName, workerOptions)

	workflowOptions := client.StartWorkflowOptions{
		ID:                              "helloworld_" + uuid.New(),
		TaskList:                        ApplicationName,
		ExecutionStartToCloseTimeout:    5 * time.Minute,
		DecisionTaskStartToCloseTimeout: 5 * time.Minute,
	}
	h.StartWorkflow(workflowOptions, "hello_world", "amar")
	select {}
}
