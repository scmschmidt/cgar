package cgroupcollect

import (
	"cgar_collect/util"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

func ReadCgroupController(cgroupPath string, controller string) map[string](string) {

	var cgroupData = make(map[string](string))

	switch controller {
	case "memory":
		for _, file := range []string{"memory.current", "memory.high", "memory.min", "memory.pressure", "memory.low", "memory.stat", "memory.swap.high", "memory.max", "memory.swap.current", "memory.swap.max"} {
			if content, err := os.ReadFile(cgroupPath + "/" + file); err == nil {
				cgroupData[file] = strings.TrimSuffix(string(content), "\n")
			} else {
				util.LogWriter.Err(err.Error())
			}
		}
	default:
		util.LogWriter.Err(fmt.Sprintf("Controller \"%s\" is currently not supported.", controller))
		return nil
	}

	return cgroupData
}

func GetFromTree(ch_data chan<- (map[string](map[string](string))), wg *sync.WaitGroup, cgroup string, depth int, controllers []string) {

	var cgroupData = make(map[string](map[string](string)))
	var controllerData = make(map[string](string))
	var child string

	cgroupPath := "/sys/fs/cgroup/" + cgroup
	cgroupData[cgroup] = make(map[string](string))

	defer wg.Done()

	// Log action.
	// util.LogWriter.Info(fmt.Sprintf("Retrieving data from \"%s\"...", cgroupPath))
	// defer util.LogWriter.Info(fmt.Sprintf("Retrieving data from \"%s\" finished.", cgroupPath))

	// We collect the cgroup data for each controller and update our cgroup map.
	for _, controller := range controllers {
		controllerData = ReadCgroupController(cgroupPath, controller)
		if controllerData != nil {
			for k, v := range controllerData {
				cgroupData[cgroup][k] = v
			}
		}
	}

	// If we have data for this cgroup, we put it into the channel.
	if len(cgroupData[cgroup]) > 0 {
		ch_data <- cgroupData
	}

	// Call GetFromTree() for each child.
	if depth > 0 {
		files, err := ioutil.ReadDir(cgroupPath)
		if err != nil {
			util.LogWriter.Err(err.Error())
		}

		for _, file := range files {
			if file.IsDir() {
				wg.Add(1)
				if cgroup != "" {
					child = cgroup + "/" + file.Name()
				} else {
					child = file.Name()
				}
				go GetFromTree(ch_data, wg, child, depth-1, controllers)
			}
		}
	}
}
