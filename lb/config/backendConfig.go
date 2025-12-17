package config

const (
    ServerMaxLoad   = 5  // how many req can a backend server handle concurrently
    Weight          = 1  // weight of each server
    CurrentLoad     = 0  // starting work load of each server
	Alive           = true // is the server alive 
)
