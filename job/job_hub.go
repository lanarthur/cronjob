/*
 * Copyright © 2018 上海讯联数据服务有限公司. All rights reserved.
 *
 * 上海讯联数据服务有限公司（以下简称“讯联数据”）拥有此源程序的完全知识产权，
 * 未经讯联数据书面授权许可，任何人或公司（不论以何种方式获得此源程序）不得擅自
 * 使用、复制、修改、二次开发、转发、售卖此源程序、相应的软件产品以及其他衍生品。
 * 否则，讯联数据将依法追究法律责任。
 *
 */

package job

import (
	"fmt"
	"github.com/CardInfoLink/coupon/config"
	"github.com/CardInfoLink/coupon/util/cron"
	"log"
)

type CronJob interface {
	// 往cron任务中注册任务单元
	Register(factory JobFactories)
	// 启动cron任务
	Start()
	// 关闭cron任务
	Close() error
	// 获取任务单元列表
	ListTask() (ret []*Task)
	// 手动执行一次任务单元，不与cron冲突，cron中相应的任务依旧会继续执行
	RunOnce(name string) error
	// 暂停cron中相应任务单元的执行
	StopTask(name string) error
	// 将已经停止的任务单元，加入执行
	RunTask(name string) error
	// 移除cron中相应的任务单元
	RemoveTask(name string) error
	// 单个任务详情
	StatusTask(name string) (*cron.JobStatus, error)
}

type cronJob struct {
	cron         *cron.Cron
	jobSpecs     map[string]*JobSpec
	jobFactories map[string]JobFactory
}

func NewCronJob(spec map[string]config.JobSpec) CronJob {
	tmp := make(map[string]*JobSpec)
	for k, v := range spec {
		tmp[k] = NewJobSpec(v.Spec)
	}
	return &cronJob{
		cron:         cron.New(),
		jobSpecs:     tmp,
		jobFactories: make(map[string]JobFactory),
	}
}

type CronConf struct {
	Name string
	Spec *JobSpec
	F    JobFactory
}

func (this *cronJob) Register(factories JobFactories) {
	for name, factory := range factories {
		this.jobFactories[name] = factory
		spec, ok := this.jobSpecs[name]
		if !ok {
			log.Printf("no cron spec of job %s in config file", name)
			continue
		}
		e := CronConf{
			Name: name,
			Spec: spec,
			F:    factory,
		}
		err := this.cron.AddFunc(e.Name, e.Spec.Spec(), func() (err error) {
			duration, period, err := e.Spec.JobPeriod()
			if err != nil {
				log.Printf("job %s spec parse with error: %s\n", e.Name, err)
			}
			log.Printf("job %s start. duration: %ds, period: %d\n", e.Name, duration, period)
			err = e.F().Run()
			if err != nil {
				log.Printf("run job %s with error: %s\n", e.Name, err)
			} else {
				log.Printf("job %s done.\n", e.Name)
			}
			return
		})
		if err != nil {
			log.Printf("failed to add job %s with error%s\n", name, err)
		}
	}

}

func (this *cronJob) Start() {
	log.Println("start cron job")
	this.cron.Start()
}

func (this *cronJob) Close() error {
	log.Println("cron job is closing")
	this.cron.Exit()
	return nil
}

type Task struct {
	Name    string      `json:"name"`
	Status  string      `json:"status"`
	HasFail bool        `json:"has_fail"`
	Err     *cron.Error `json:"err"`
}

// task为任务单元，多个任务单元组成任务,
func (this *cronJob) ListTask() (ret []*Task) {
	tasks := this.cron.Entries()
	for _, v := range tasks {
		task := &Task{}
		task.Name = v.Name
		task.HasFail = v.HasFail
		task.Err = v.Err
		task.Status = v.Status
		ret = append(ret, task)
	}
	return
}

// 手动执行失败一次的任务
func (this *cronJob) RunOnce(name string) error {
	jobFactory, ok := this.jobFactories[name]
	if !ok {
		return fmt.Errorf("this job: %s did not register before", name)
	}
	job := jobFactory()

	err := job.Run()
	return err
}
func (this *cronJob) StopTask(name string) error {
	_, ok := this.jobFactories[name]
	if !ok {
		return fmt.Errorf("this job: %s did not register before", name)
	}
	this.cron.StopJob(name)
	return nil
}

// 将已经停止的task重新开始执行
func (this *cronJob) RunTask(name string) error {
	_, ok := this.jobFactories[name]
	if !ok {
		return fmt.Errorf("this job: %s did not register before", name)
	}
	this.cron.RunJob(name)
	return nil
}

func (this *cronJob) RemoveTask(name string) error {
	_, ok := this.jobFactories[name]
	if !ok {
		return fmt.Errorf("this job: %s did not register before", name)
	}
	this.cron.RemoveJob(name)
	return nil
}
func (this *cronJob) StatusTask(name string) (*cron.JobStatus, error) {
	_, ok := this.jobFactories[name]
	if !ok {
		return nil, fmt.Errorf("this job: %s did not register before", name)
	}
	return this.cron.JobStatus(name), nil
}
