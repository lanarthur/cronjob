package cron

import (
	"log"
	"runtime"
	"sort"
	"time"
)

const (
	statusRun  = "running"
	statusStop = "stopped"
)

const timeFormatLayout = "2006-01-02 15:04:05"

// Error represents a err during job executing
type Error struct {
	At  string // fail at
	Msg error  // fail reason
}

func newError(err error) (this *Error) {
	this = &Error{}
	now := time.Now().Format(timeFormatLayout)
	this.At = now
	this.Msg = err
	return
}

// Cron keeps track of any number of running, invoking the associated func as
// specified by the schedule. It may be started, stopped, and the running may
// be inspected while running.
type Cron struct {
	running  []*job
	stopped  []*job
	add      chan *job
	stop     chan string
	remove   chan string
	exit     chan struct{}
	status   string
	ErrorLog *log.Logger
	location *time.Location
}

// Job is an interface for submitted cron jobs.
type Job interface {
	Run() (err error)
}

// The Schedule describes a job's duty cycle.
type Schedule interface {
	// Return the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}

// job consists of a schedule and the func to execute on that schedule.
type job struct {
	Name    string
	HasFail bool
	Err     *Error
	Status  string

	// The schedule on which this job should be run.
	Schedule Schedule

	// The next time the job will run. This is the zero time if Cron has not been
	// started or this entry's schedule is unsatisfiable
	Next time.Time

	// The last time this job was run. This is the zero time if the job has never
	// been run.
	Prev time.Time

	// The Job to run.
	Job Job
}

// byTime is a wrapper for sorting the entry array by time
// (with zero time at the end).
type byTime []*job

func (s byTime) Len() int      { return len(s) }
func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byTime) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	if s[i].Next.IsZero() {
		return false
	}
	if s[j].Next.IsZero() {
		return true
	}
	return s[i].Next.Before(s[j].Next)
}

// New returns a new Cron job runner, in the Local time zone.
func New() *Cron {
	return NewWithLocation(time.Now().Location())
}

// NewWithLocation returns a new Cron job runner.
func NewWithLocation(location *time.Location) *Cron {
	return &Cron{
		running:  nil,
		stopped:  nil,
		add:      make(chan *job),
		stop:     make(chan string),
		remove:   make(chan string),
		exit:     make(chan struct{}),
		status:   statusStop,
		ErrorLog: nil,
		location: location,
	}
}

// FuncJob A wrapper that turns a func() into a cron.Job
type FuncJob func() (err error)

// Run implements run interface
func (f FuncJob) Run() (err error) {
	err = f()
	return
}

// AddFunc adds a func to the Cron to be run on the given schedule.
func (c *Cron) AddFunc(name, spec string, cmd func() (err error)) error {
	return c.AddJob(name, spec, FuncJob(cmd))
}

// AddJob adds a Job to the Cron to be run on the given schedule.
func (c *Cron) AddJob(name, spec string, cmd Job) error {
	schedule, err := Parse(spec)
	if err != nil {
		return err
	}
	c.Schedule(name, schedule, cmd)
	return nil
}

// RemoveJob remove job from cron accord job name
func (c *Cron) RemoveJob(name string) {
	for k, v := range c.stopped {
		if v.Name == name {
			c.stopped = append(c.stopped[:k], c.stopped[k+1:]...)
			return
		}
	}
	c.remove <- name
	return
}

// JobStatus represents a job info
type JobStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	HasFail bool   `json:"has_fail"`
	Err     *Error `json:"err"`
}

// JobStatus return job info accord job name
func (c *Cron) JobStatus(name string) (ret *JobStatus) {
	ret = &JobStatus{}
	for _, v := range c.running {
		if v.Name == name {
			ret.Name = v.Name
			ret.Status = statusRun
			ret.HasFail = v.HasFail
			ret.Err = v.Err
			return
		}
	}
	for _, v := range c.stopped {
		if v.Name == name {
			ret.Name = v.Name
			ret.Status = statusStop
			ret.HasFail = v.HasFail
			ret.Err = v.Err
			return
		}
	}
	return
}

// StopJob stop job accord job name
func (c *Cron) StopJob(name string) {
	c.stop <- name
	return
}

// RunJob move stopped job to running accorded job name
func (c *Cron) RunJob(name string) {
	var e *job
	for k, v := range c.stopped {
		if v.Name == name {
			e = &job{
				Name:     v.Name,
				HasFail:  v.HasFail,
				Err:      v.Err,
				Job:      v.Job,
				Next:     v.Next,
				Prev:     v.Prev,
				Schedule: v.Schedule,
			}
			c.stopped = append(c.stopped[:k], c.stopped[k+1:]...)
			break
		}
	}
	if e == nil {
		return
	}
	if c.status == statusStop {
		c.running = append(c.running, e)
		go c.run()
		return
	}
	c.add <- e
}

// Schedule adds a Job to the Cron to be run on the given schedule.
func (c *Cron) Schedule(name string, schedule Schedule, cmd Job) error {
	entry := &job{
		Name:     name,
		HasFail:  false,
		Err:      &Error{},
		Schedule: schedule,
		Job:      cmd,
	}
	if c.status == statusStop {
		c.running = append(c.running, entry)
		go c.run()
		return nil
	}

	c.add <- entry
	return nil
}

// Jobs returns a snapshot of the cron running.
func (c *Cron) Jobs() []*job {
	return c.jobSnapshot()
}

// Location gets the time zone location
func (c *Cron) Location() *time.Location {
	return c.location
}

// Start the cron scheduler in its own go-routine, or no-op if already started.
func (c *Cron) Start() {
	if c.status == statusRun {
		return
	}
	c.status = statusRun
	c.running = append(c.running, c.stopped...)
	c.stopped = []*job{}
	go c.run()
}

// Exit stops the cron scheduler if it is running; otherwise it does nothing.
func (c *Cron) Exit() {
	if c.status == statusStop {
		return
	}
	c.exit <- struct{}{}
	c.status = statusStop
	c.stopped = append(c.stopped, c.running...)
	c.running = []*job{}
}

// Status return cronjob status
func (c *Cron) Status() string {
	return c.status
}

func (c *Cron) runWithRecovery(e *job) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.logf("cron: panic running job: %v\n%s", r, buf)
		}
	}()
	err := e.Job.Run()
	if err != nil {
		e.Err = newError(err)
		e.HasFail = true
	}
	return
}

// Run the scheduler. this is private just due to the need to synchronize
// access to the 'running' state variable.
func (c *Cron) run() {
	// Figure out the next activation times for each entry.
	now := c.now()
	for _, entry := range c.running {
		entry.Next = entry.Schedule.Next(now)
	}

	for {
		// Determine the next entry to run.
		sort.Sort(byTime(c.running))

		var timer *time.Timer
		if len(c.running) == 0 || c.running[0].Next.IsZero() {
			// If there are no running yet, just sleep - it still handles new running
			// and exit requests.
			timer = time.NewTimer(100000 * time.Hour)
		} else {
			timer = time.NewTimer(c.running[0].Next.Sub(now))
		}

		for {
			select {
			case now = <-timer.C:
				now = now.In(c.location)
				// Run every entry whose next time was less than now
				for _, e := range c.running {

					if e.Next.After(now) || e.Next.IsZero() {
						break
					}
					e := e
					go c.runWithRecovery(e)
					e.Prev = e.Next
					e.Next = e.Schedule.Next(now)

				}

			case newEntry := <-c.add:
				timer.Stop()
				now = c.now()
				newEntry.Next = newEntry.Schedule.Next(now)
				c.running = append(c.running, newEntry)

			case stopName := <-c.stop:
				timer.Stop()
				for k, v := range c.running {
					if v.Name == stopName {
						c.stopped = append(c.stopped, v)
						c.running = append(c.running[:k], c.running[k+1:]...)
						break
					}
				}

			case removeName := <-c.remove:
				timer.Stop()
				for k, v := range c.running {
					if v.Name == removeName {
						c.running = append(c.running[:k], c.running[k+1:]...)
						break
					}
				}
				for k, v := range c.stopped {
					if v.Name == removeName {
						c.stopped = append(c.stopped[:k], c.stopped[k+1:]...)
						break
					}
				}

			case <-c.exit:
				timer.Stop()
				return
			}

			break
		}
	}
}

func (c *Cron) logf(format string, args ...interface{}) {
	if c.ErrorLog != nil {
		c.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (c *Cron) jobSnapshot() []*job {
	var entries []*job
	for _, e := range c.running {
		entries = append(entries, &job{
			Name:     e.Name,
			Status:   statusRun,
			HasFail:  e.HasFail,
			Err:      e.Err,
			Schedule: e.Schedule,
			Next:     e.Next,
			Prev:     e.Prev,
			Job:      e.Job,
		})
	}
	for _, e := range c.stopped {
		entries = append(entries, &job{
			Name:     e.Name,
			Status:   statusStop,
			HasFail:  e.HasFail,
			Err:      e.Err,
			Schedule: e.Schedule,
			Next:     e.Next,
			Prev:     e.Prev,
			Job:      e.Job,
		})
	}
	return entries
}

func (c *Cron) now() time.Time {
	return time.Now().In(c.location)
}
