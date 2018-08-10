package example

import (
	"fmt"
	"github.com/lanarthur/cronjob/job"
	"time"
)

const Jake = "jake"

type jake struct {
	name string
}

func NewJakeJob(name string) job.Job {
	return &jake{name}
}

func (this jake) Run() (err error) {
	fmt.Printf("%s ---, now:%s\n", this.name, time.Now().String())
	return
}

func (this jake) RunWithTask() (err error) { return nil }
