// This file is kept to ensure backward compatibility.

package boomer

import (
	"flag"
	"log"
	"math"
	"time"
)

var masterHost string
var masterPort int
var maxRPS int64
var requestIncreaseRate string
var runTasks string
var memoryProfile string
var memoryProfileDuration time.Duration
var cpuProfile string
var cpuProfileDuration time.Duration

func createRateLimiter(maxRPS int64, requestIncreaseRate string) (rateLimiter RateLimiter, err error) {
	if requestIncreaseRate != "-1" {
		if maxRPS > 0 {
			log.Println("The max RPS that boomer may generate is limited to", maxRPS, "with a increase rate", requestIncreaseRate)
			rateLimiter, err = NewRampUpRateLimiter(maxRPS, requestIncreaseRate, time.Second)
		} else {
			log.Println("The max RPS that boomer may generate is limited by a increase rate", requestIncreaseRate)
			rateLimiter, err = NewRampUpRateLimiter(math.MaxInt64, requestIncreaseRate, time.Second)
		}
	} else {
		if maxRPS > 0 {
			log.Println("The max RPS that boomer may generate is limited to", maxRPS)
			rateLimiter = NewStableRateLimiter(maxRPS, time.Second)
		}
	}
	return rateLimiter, err
}

func init() {
	flag.Int64Var(&maxRPS, "max-rps", 0, "Max RPS that boomer can generate, disabled by default.")
	flag.StringVar(&requestIncreaseRate, "request-increase-rate", "-1", "Request increase rate, disabled by default.")
	flag.StringVar(&runTasks, "run-tasks", "", "Run tasks without connecting to the master, multiply tasks is separated by comma. Usually, it's for debug purpose.")
	flag.StringVar(&masterHost, "master-host", "127.0.0.1", "Host or IP address of locust master for distributed load testing.")
	flag.IntVar(&masterPort, "master-port", 8080, "The port to connect to that is used by the locust master for distributed load testing.")
	flag.StringVar(&memoryProfile, "mem-profile", "", "Enable memory profiling.")
	flag.DurationVar(&memoryProfileDuration, "mem-profile-duration", 30*time.Second, "Memory profile duration.")
	flag.StringVar(&cpuProfile, "cpu-profile", "", "Enable CPU profiling.")
	flag.DurationVar(&cpuProfileDuration, "cpu-profile-duration", 30*time.Second, "CPU profile duration.")
}
