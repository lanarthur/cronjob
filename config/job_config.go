package config

type JobSpec struct {
	Spec string
}

type JobConf struct {
	CronJob        map[string]*JobSpec
}

var Cfg     *JobConf

func init() {

}