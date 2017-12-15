package services

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/tracker"
)

type builder struct {
	Cfg       OperationConfig
	Overrides ContextOverride
}

func NewBuilder(cfg OperationConfig, overrides ContextOverride) *builder {
	return &builder{
		Cfg:       cfg,
		Overrides: overrides,
	}
}

func (b *builder) Build(task tracker.Task, service ...*ServiceConfig) error {
	for _, service := range service {
		err := b.doBuild(task, service)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (b *builder) doBuild(task tracker.Task, service *ServiceConfig) error {
	if b.Cfg.IsExcluded(service) {
		return nil
	}
	return errors.WithStack(b.BuildSync(task, service, false))
}

// BuildSync will buid the service synchronously.
// If force is false, the build will be skipped if the service is already running.
func (b *builder) BuildSync(task tracker.Task, service *ServiceConfig, force bool) error {
	name := service.GetName()
	t := task.Child(name)
	return errors.WithStack(b.BuildWithTracker(t, service, force))
}

// BuildWithTracker builds a service.
// If force is false, the build will be skipped if the service is already running.
func (b *builder) BuildWithTracker(task tracker.Task, service *ServiceConfig, force bool) error {
	if service.Commands.Build == "" {
		return nil
	}
	if task == nil {
		return errors.New("task is nil")
	}
	job := task.Child("Build")
	job.SetState(tracker.TaskStateInProgress)

	c, err := service.GetCommand(b.Overrides)
	if err != nil {
		return errors.WithStack(err)
	}
	if !force && c.Pid != 0 {
		job.SetState(tracker.TaskStateWarning, "Already running")
		return nil
	}

	cmd, err := commandline.ConstructCommand(b.Cfg.WorkingDir, service.Path, service.Commands.Build, c.Getenv)
	if err != nil {
		job.SetState(tracker.TaskStateFailed, err.Error())
		return errors.WithStack(err)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		job.SetState(tracker.TaskStateFailed, err.Error(), string(out))
		return errors.WithMessage(err, "running build command")
	}

	job.SetState(tracker.TaskStateSuccess)
	return nil
}
