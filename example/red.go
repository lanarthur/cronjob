package example

import (
	"errors"
	"fmt"
	"github.com/lanarthur/cronjob/job"
	"time"
)

const Red = "red"

type red struct {
	name string
}

func NewRedJob(name string) job.Job {
	return &red{name}
}

func (this red) Run() (err error) {
	fmt.Printf("%s ---, now:%s\n", this.name, time.Now().String())
	err = errors.New("err from red")
	return
}

func (this red) RunWithTask() (err error) { return nil }
