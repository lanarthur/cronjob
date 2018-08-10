package main

import (
	"time"

	"github.com/lanarthur/cronjob/config"
	"github.com/lanarthur/cronjob/example"
	"github.com/lanarthur/cronjob/job"
)

func main() {
	cfg := config.Config()
	cronJob := job.NewCronJob(cfg.CronJob)
	job.Factories[example.Red] = func() job.Job {
		return example.NewRedJob(example.Red)
	}
	job.Factories[example.Mike] = func() job.Job {
		return example.NewMikeJob(example.Mike)
	}
	job.Factories[example.Jake] = func() job.Job {
		return example.NewJakeJob(example.Jake)
	}
	cronJob.Register(job.Factories)
	cronJob.Start()
	time.Sleep(time.Duration(30) * time.Second)
}
