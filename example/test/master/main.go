package main

import (
	"boomer"
	"context"
	"github.com/rs/zerolog/log"
)

func main() {
	var (
		expectWorkers                = 1
		expectWorkersMaxWait         = 60 * 3  // 秒速
		spawnCount           int64   = 1       // 总数
		spawnRate            float64 = 1       // 速率
		runTime              int64   = 100 * 3 // 速率
	)

	log.Logger = log.With().Caller().Logger()

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

	masterBoomer.RunMaster()
}
