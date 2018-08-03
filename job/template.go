package job


type Job interface {
	// cron任务实现
	Run() (err error)
	// 异步任务实现
	RunWithTask() (err error)
}

type JobFactory func() Job

type JobFactories map[string]JobFactory

var Factories JobFactories = make(map[string]JobFactory)