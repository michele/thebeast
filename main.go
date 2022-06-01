package main

import (
	"thebeast/cmd"
	"thebeast/configuration"
	"thebeast/utils"

	"runtime"
)

var batchSize int

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	batchSize = runtime.NumCPU() * configuration.Config.BatchPerCore

	utils.InitLogger().Printf("Loading app %s in %s", configuration.Config.AppName, configuration.Config.GoEnv)
	utils.InitLogger().Printf("Max Procs: %d; Batch Size: %d", runtime.NumCPU(), batchSize)

	cmd.BatchSize = batchSize
	cmd.Execute()

}
