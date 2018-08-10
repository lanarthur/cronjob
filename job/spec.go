package job

import (
	"github.com/lanarthur/cronjob/cron"
	"math"
	"time"
)

type JobSpec struct {
	spec string
	sche cron.Schedule
}

func NewJobSpec(spec string) *JobSpec {
	return &JobSpec{
		spec: spec,
	}
}

func (s JobSpec) Spec() string {
	return s.spec
}

func (s JobSpec) schedule() (cron.Schedule, error) {
	return cron.Parse(s.spec)
}

func (s JobSpec) JobPeriod() (duration, period int64, err error) {
	if s.sche == nil {
		s.sche, err = s.schedule()
		if err != nil {
			return
		}
	}
	now := time.Now()
	seconds := float64(s.sche.Next(now).Sub(now)) / float64(time.Second)
	duration = int64(math.Ceil(seconds))
	period = int64(math.Ceil(float64(now.Unix()) / float64(duration)))
	return
}
