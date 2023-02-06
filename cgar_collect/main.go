/*
	cgar_collect

	Collects cgroup data and writes them to a log file.


	If new controllers need to be added:
	------------------------------------
		`ReadCgroupController()` (`cgroupcollect/cgroupcollect.go`) needs to get enhanced. That's all.

	ToDo:
	----

	Changelog:
	----------
		05.02.2023      v0.1        - first implementation
*/

package main

import (
	"cgar_collect/cgroupcollect"
	"cgar_collect/config"
	"cgar_collect/util"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

func build_entry(ch_data <-chan (map[string](map[string](string))), ch_result chan<- map[string](map[string](map[string](string))), wg_result *sync.WaitGroup) {

	var entry = make(map[string](map[string](map[string](string))))

	defer wg_result.Done()

	timestamp := time.Now().Format(time.RFC3339)
	entry[timestamp] = make((map[string](map[string](string))))
	for cgroupData := range ch_data {
		for k, v := range cgroupData {
			entry[timestamp][k] = v
		}
	}
	ch_result <- entry

}

func main() {

	var ch_data = make(chan map[string](map[string](string)), 100)
	var ch_result = make(chan map[string](map[string](map[string](string))), 1)

	var wg_data sync.WaitGroup
	var wg_result sync.WaitGroup

	var configFile string

	// Initial Logging.
	util.LogWriter.Info(fmt.Sprintf("Called as: %s", strings.Join(os.Args, " ")))
	defer util.LogWriter.Info("Terminated.")

	// Read configuration (either from given path or default config file).
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	} else {
		configFile = "/etc/cgar/conf.json"
	}
	config, err := config.LoadConfig(configFile)
	if err != nil {
		util.LogWriter.Crit(err.Error())
		util.LogWriter.Info("Terminated.")
		os.Exit(2)
	}

	// Start reading cgroup data concurrently.
	for _, collection := range config.Collect {
		fmt.Println(collection.Cgroup, collection.Depth, collection.Controllers)
		wg_data.Add(1)
		go cgroupcollect.GetFromTree(ch_data, &wg_data, collection.Cgroup, collection.Depth, collection.Controllers)
	}

	// Start collecting results concurrently.
	wg_result.Add(1)
	go build_entry(ch_data, ch_result, &wg_result)

	// Wait for all collectors to finish.
	wg_data.Wait()
	close(ch_data)

	// Wait for the result builder to finish.
	wg_result.Wait()
	close(ch_result)

	// Convert result into a JSON string add append to the log.
	jsonStr, err := json.Marshal(<-ch_result)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
	} else {
		f, err := os.OpenFile(config.Logfile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			util.LogWriter.Err(err.Error())
		} else {
			defer f.Close()
			if _, err := f.WriteString(string(jsonStr) + "\n"); err != nil {
				util.LogWriter.Err(err.Error())
			}
		}
	}

}
