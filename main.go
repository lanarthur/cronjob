package main

import (
	"github.com/lanarthur/cronjob/example"
	"github.com/lanarthur/cronjob/config"
	"github.com/lanarthur/cronjob/job"
)

func main() {
	cronJob := job.NewCronJob(config.Cfg.CronJob)
	job.Factories[example.Red] = func() job.Job {
		return example.NewRedJob(example.Red)
	}
	cronJob.Register(job.Factories)
	cronJob.Start()
}