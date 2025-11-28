package config

const (
	HighWeight = 7 // percentage of high priority req to be selected
	LowWeight  = 3 // percentage of low priority req to be selected
	SleepTime  = 5 // time to sleep (ms)
	SleepTimeMaxConcurrency = 10 // time to wait if at max concurrency (ms)
	SleepTimeQueue = 50 // time to wait for queuing (ms)
)