package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

type JobSpec struct {
	Spec string
}

type JobConf struct {
	CronJob map[string]JobSpec `toml:"cron_job"`
}

var (
	cfg     *JobConf
	cfgLock sync.RWMutex
	once    sync.Once
)

// 单例
func Config() *JobConf {
	once.Do(reloadCfg)
	cfgLock.RLock()
	defer cfgLock.RUnlock()
	return cfg
}

// 任务配置文件叫job.toml
func reloadCfg() {
	config := new(JobConf)
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	if _, err := toml.DecodeFile(dir+"/job.toml", &config); err != nil {
		panic(err)
	}
	cfg = config
	return
}
