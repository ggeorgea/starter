package common

import (
	// "fmt"
	"encoding/json"
	"io/ioutil"
	"regexp"
)

// Looks for node version in the package.json. If found returns true, version if not false, ""
func GetNodeVersion(packageJsonFile string) (bool, string) {
	buf, err := ioutil.ReadFile(packageJsonFile)
	if err != nil {
		return false, err.Error()
	}

	var data map[string](interface{})
	err = json.Unmarshal(buf, &data)

	if err != nil {
		return false, err.Error()
	}

	if data["engines"] == nil {
		return false, ""
	}

	if nodeVersion, ok := data["engines"].(map[string]interface{})["node"].(string); ok {
		re := regexp.MustCompile("(\\d).(\\d).*")
		nodeVersion = re.FindStringSubmatch(nodeVersion)[1] + "." + re.FindStringSubmatch(nodeVersion)[2]
		if re.FindStringSubmatch(nodeVersion)[2] == "0" {
			nodeVersion = re.FindStringSubmatch(nodeVersion)[1]
		}
		return true, nodeVersion
	} else {
		return false, ""
	}

}

func GetNodeDatabase(packageJsonFile string, databaseNames ...string) (bool, string) {
	found, name := GetDependencyVersion(packageJsonFile, databaseNames...)
	return found, name
}

func GetDependencyVersion(packageJsonFile string, dependencyNames ...string) (bool, string) {
	buf, err := ioutil.ReadFile(packageJsonFile)
	if err != nil {
		return false, err.Error()
	}

	var data map[string](interface{})
	err = json.Unmarshal(buf, &data)

	if err != nil {
		return false, err.Error()
	}

	for dependency, version := range data["dependencies"].(map[string]interface{}) {
		for _, dependencyName := range dependencyNames {
			found := dependencyName == dependency
			if found {
				return true, version.(string)
			}
		}

	}

	return false, ""
}

func GetScriptsStart(packageJsonFile string) (bool, string) {
	buf, err := ioutil.ReadFile(packageJsonFile)
	if err != nil {
		return false, err.Error()
	}

	var data map[string](interface{})
	err = json.Unmarshal(buf, &data)

	if err != nil {
		return false, err.Error()
	}

	if data["scripts"] == nil {
		return false, ""
	}

	if start, ok := data["scripts"].(map[string]interface{})["start"].(string); ok {
		return true, start
	} else {
		return false, ""
	}
}
