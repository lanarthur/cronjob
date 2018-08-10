package example

import (
	"fmt"
	"github.com/lanarthur/cronjob/job"
	"time"
)

const Mike = "mike"

type mike struct {
	name string
}

func NewMikeJob(name string) job.Job {
	return &mike{name}
}

func (this mike) Run() (err error) {
	fmt.Printf("%s ---, now:%s\n", this.name, time.Now().String())
	return
}

func (this mike) RunWithTask() (err error) { return nil }
