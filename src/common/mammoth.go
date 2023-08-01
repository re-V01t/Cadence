package common

import (
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/encoded"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func buildLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.Level.SetLevel(zapcore.InfoLevel)

	var err error
	logger, err := config.Build()
	if err != nil {
		panic("Failed to setup logger")
	}

	return logger
}

type (
	Configuration struct {
		DomainName      string `yaml:"domain"`
		ServiceName     string `yaml:"service"`
		HostNameAndPort string `yaml:"host"`
		//Prometheus      *prometheus.Configuration `yaml:"prometheus"`
	}

	registryOption struct {
		registry interface{}
		alias    string
	}

	Helper struct {
		Builder            *WorkflowClientBuilder
		Client             client.Client
		Service            workflowserviceclient.Interface
		DataConverter      encoded.DataConverter
		Logger             *zap.Logger
		Config             *Configuration
		WorkerMetricScope  tally.Scope
		ServiceMetricScope tally.Scope
		Tracer             opentracing.Tracer
		CtxPropagators     []workflow.ContextPropagator
		workflowRegistries []registryOption
		activityRegistries []registryOption
	}
)

func (h *Helper) RegisterWorkflow(workflow interface{}) {
	h.RegisterWorkflowWithAlias(workflow, "")
}

func (h *Helper) RegisterWorkflowWithAlias(workflow interface{}, alias string) {
	registryOption := registryOption{
		registry: workflow,
		alias:    alias,
	}
	h.workflowRegistries = append(h.workflowRegistries, registryOption)
}

func (h *Helper) RegisterActivity(activity interface{}) {
	h.RegisterActivityWithAlias(activity, "")
}

func (h *Helper) RegisterActivityWithAlias(activity interface{}, alias string) {
	registryOption := registryOption{
		registry: activity,
		alias:    alias,
	}
	h.activityRegistries = append(h.activityRegistries, registryOption)
}

func (h *Helper) Setup(configuration *Configuration) {
	if h.Service != nil {
		return
	}
	h.Logger = buildLogger()
	h.ServiceMetricScope = tally.NoopScope
	h.WorkerMetricScope = tally.NoopScope
	h.Config = configuration
	h.Builder = NewBuilder(h.Logger).
		SetHostPort(h.Config.HostNameAndPort).
		SetDomain(h.Config.DomainName).
		//SetMetricsScope(h.ServiceMetricScope).
		SetDataConverter(h.DataConverter).
		//SetTracer(h.Tracer).
		SetContextPropagators(h.CtxPropagators)
	fmt.Println("reached - 1")
	service, err := h.Builder.BuildServiceClient()
	if err != nil {
		panic(err)
	}
	fmt.Println("reached - 2")
	h.Service = service
	h.Client, err = h.Builder.BuildCadenceClient()
	if err != nil {
		h.Logger.Error("Failed to build cadence client.", zap.Error(err))
		panic(err)
	}
	fmt.Println("reached - 3")
	//domainClient, _ := h.Builder.BuildCadenceDomainClient()
	//var domainName = fmt.Sprintf("%v", h.Config.DomainName)
	//registerDomainRequest := shared.Default_RegisterDomainRequest()
	//registerDomainRequest.Name = &domainName
	//fmt.Println("HERE!")
	//err = domainClient.Register(context.Background(), registerDomainRequest)
	//if err != nil {
	//	h.Logger.Info("Something went wrong", zap.String("Domain", h.Config.DomainName), zap.Error(err))
	//	panic(err) // TODO remove this after
	//} else {
	//	h.Logger.Info("Domain successfully registered.", zap.String("Domain", h.Config.DomainName))
	//}
	//_, err = domainClient.Describe(context.Background(), h.Config.DomainName)
	//if err != nil {
	//	h.Logger.Info("Domain doesn't exist", zap.String("Domain", h.Config.DomainName), zap.Error(err))
	//} else {
	//	h.Logger.Info("Domain successfully registered.", zap.String("Domain", h.Config.DomainName))
	//}

}

func (h *Helper) registerWorkflowAndActivity(worker worker.Worker) {
	for _, w := range h.workflowRegistries {
		if len(w.alias) == 0 {
			worker.RegisterWorkflow(w.registry)
		} else {
			worker.RegisterWorkflowWithOptions(w.registry, workflow.RegisterOptions{Name: w.alias})
		}
	}
	for _, act := range h.activityRegistries {
		if len(act.alias) == 0 {
			worker.RegisterActivity(act.registry)
		} else {
			worker.RegisterActivityWithOptions(act.registry, activity.RegisterOptions{Name: act.alias})
		}
	}
}

// StartWorkers starts workflow worker and activity worker based on configured options.
func (h *Helper) StartWorkers(domainName string, groupName string, options worker.Options) {
	worker := worker.New(h.Service, domainName, groupName, options)
	h.registerWorkflowAndActivity(worker)

	err := worker.Start()
	if err != nil {
		h.Logger.Error("Failed to start workers.", zap.Error(err))
		panic("Failed to start workers")
	}
}

// StartWorkflow starts a workflow
func (h *Helper) StartWorkflow(
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) *workflow.Execution {
	return h.StartWorkflowWithCtx(context.Background(), options, workflow, args...)
}

// StartWorkflowWithCtx starts a workflow with the provided context
func (h *Helper) StartWorkflowWithCtx(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) *workflow.Execution {
	we, err := h.Client.StartWorkflow(ctx, options, workflow, args...)
	if err != nil {
		h.Logger.Error("Failed to create workflow", zap.Error(err))
		panic("Failed to create workflow.")
	} else {
		h.Logger.Info("Started Workflow", zap.String("WorkflowID", we.ID), zap.String("RunID", we.RunID))
		return we
	}
}
