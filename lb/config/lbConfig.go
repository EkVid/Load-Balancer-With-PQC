package config

const (
	MaxRetry = 10 // maximum tries to spin up new backend server
	WaitTime = 1 // time to wait between retries (sec)
	ReqLimitScaleUp = 1 // limit to decide if need to scale up 
	NumSeversToAdd = 3 // how many servers to add if needed
	MinServers = 5 // how many servers we should keep minimum
	ClientTimeOut = 5 // how long client will wait for server to respond (sec)
)