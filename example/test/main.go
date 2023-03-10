package main

import (
	"boomer"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"math/rand"
	"sync"
	"time"
)

func main() {
	var (
		expectWorkers                = 1
		expectWorkersMaxWait         = 60 * 3  // 秒速
		spawnCount           int64   = 1       // 总数
		spawnRate            float64 = 1       // 速率
		runTime              int64   = 100 * 3 // 速率
	)

	log.Logger = log.With().Caller().Logger().Level(zerolog.ErrorLevel)

	profile := &boomer.Profile{
		SpawnCount:               spawnCount,
		SpawnRate:                spawnRate,
		RunTime:                  runTime,
		MaxRPS:                   0,
		LoopCount:                0,
		RequestIncreaseRate:      "",
		MemoryProfile:            "",
		MemoryProfileDuration:    0,
		CPUProfile:               "",
		CPUProfileDuration:       0,
		PrometheusPushgatewayURL: "http://10.18.97.253:9091/",
		DisableConsoleOutput:     false,
		DisableCompression:       false,
		DisableKeepalive:         true,
	}
	masterHost := "127.0.0.1"
	masterPort := 8080
	masterBoomer := boomer.NewMasterBoomer(masterHost, masterPort)
	masterBoomer.EnableGracefulQuit(context.Background())
	masterBoomer.SetProfile(profile)
	masterBoomer.AddOutput(boomer.NewPrometheusPusherOutput(profile.PrometheusPushgatewayURL, "boomer-grpc-1", "standalone"))
	masterBoomer.SetAutoStart()
	masterBoomer.SetDisableKeepAlive(true)
	masterBoomer.SetExpectWorkers(expectWorkers, expectWorkersMaxWait)
	masterBoomer.SetSpawnCount(spawnCount)
	masterBoomer.SetSpawnRate(spawnRate)
	masterBoomer.SetRunTime(runTime)

	go masterBoomer.RunMaster()

	initTask := &boomer.Task{
		Weight: 1,
		Name:   "taskInit",
		Fn: func(ctx boomer.Context) {
			fmt.Println("task init")
			boomer.RecordFailure("tcp", "task-2", rand.Int63n(30)+50, "test exception")

		},
	}
	var tasks []*boomer.Task

	task1 := &boomer.Task{
		Weight: 1,
		Name:   "task1",
		Fn: func(ctx boomer.Context) {
			time.Sleep(1 * time.Second)
			fmt.Println("task 1")
			//boomer.RecordSuccess("tcp", "task-1", rand.Int63n(50)+100, rand.Int63n(100))
			boomer.RecordFailure("tcp", "task-2", rand.Int63n(30)+50, "test exception")

		},
	}

	task2 := &boomer.Task{
		Weight: 2,
		Name:   "task1",
		Fn: func(ctx boomer.Context) {
			time.Sleep(1 * time.Second)

			fmt.Println("task 2")
			boomer.RecordFailure("tcp", "task-2", rand.Int63n(30)+50, "test exception")
			//boomer.Quit()
			//time.Sleep(1 * time.Hour)
		},
	}

	tasks = append(tasks, task1, task2)

	wg := sync.WaitGroup{}
	wg.Add(expectWorkers)
	for i := 0; i < expectWorkers; i++ {
		go func() {
			boomer.SetInitTask(initTask)
			boomer.Run(tasks...)
			wg.Done()
		}()
	}

	wg.Wait()

}
