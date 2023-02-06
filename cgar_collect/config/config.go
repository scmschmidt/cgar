package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Collections struct {
	Cgroup      string
	Depth       int
	Controllers []string
}

type cgarConfig struct {
	Logfile string
	Collect []Collections
}

func LoadConfig(filename string) (cgarConfig, error) {

	var configuration cgarConfig

	configFile, err := os.Open(filename)
	if err != nil {
		return configuration, fmt.Errorf("Error opening \"%s\": %s", filename, err.Error())
	}
	defer configFile.Close()

	cfgJson, err := ioutil.ReadAll(configFile)
	if err != nil {
		return configuration, fmt.Errorf("Error reading \"%s\": %s", filename, err.Error())
	}

	if err := json.Unmarshal([]byte(cfgJson), &configuration); err != nil {
		return configuration, fmt.Errorf("Error parsing \"%s\": %s", filename, err.Error())
	}

	return configuration, nil
}
