package example

import (
	"fmt"
	"github.com/lanarthur/cronjob/job"
	"errors"
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
	//time.Sleep(time.Duration(3)*time.Second)
	fmt.Printf("%s ---, now:%s\n", this.name, time.Now().String())
	err = errors.New("err from red")
	return
}

func (this red) RunWithTask() (err error) { return nil }
