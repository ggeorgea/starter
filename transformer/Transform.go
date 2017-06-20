package transformer

import (
	"bufio"
	"strings"
	"fmt"
	"os"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"log"

	"github.com/cloud66/starter/common"

)


//main transformation format function
func Transformer(filename string, formatTarget string, directlyTransformed bool) (bool, error) {

	var err error
	_, err = os.Stat(formatTarget)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(formatTarget)
		checkError(err)
		defer file.Close()
	} else {
		common.PrintError("File %s already exists. Will be overwritten.\n", formatTarget)
	}

	//HANDLE ENV_VARS
	auxFile, dockerText, initialDockerText := accomodateEnvVars(filename)
	dockerComp, _ := os.Create(auxFile)
	_, err = dockerComp.WriteString(dockerText)

	err = dockerComp.Sync()
	checkError(err)

	dockerComp.Close()

	yamlFile, err := ioutil.ReadFile(auxFile)

	dockerCompose := Docker_compose{
		Services: make(map[string]docker_Service),
		Version:  "",
	}

	serviceYaml := Serviceyml{
		Services: make(map[string]common.Service),
		Dbs:      make([]string, 0),
	}

	if err := yaml.Unmarshal([]byte(yamlFile), &dockerCompose); err != nil {
		fmt.Println(err.Error())
	}

	if len(dockerCompose.Services) == 0 {
		d := make(map[string]docker_Service)
		err = yaml.Unmarshal([]byte(yamlFile), &d)
		checkError(err)

		serviceYaml.Services, serviceYaml.Dbs = copyToServiceYML(d, directlyTransformed)

	} else {

		serviceYaml.Services, serviceYaml.Dbs = copyToServiceYML(dockerCompose.Services, directlyTransformed)
	}
	if len(serviceYaml.Dbs) != 0 {
		if serviceYaml.Dbs[len(serviceYaml.Dbs)-1] == "" {
			serviceYaml.Dbs = serviceYaml.Dbs[:len(serviceYaml.Dbs)-1]
		}
	}

	file, err := yaml.Marshal(serviceYaml)

	err = ioutil.WriteFile("service.yml", file, 0644)
	if err != nil {
		log.Fatalf("ioutil.WriteFile: %v", err)
	}

	service_yml, er := os.OpenFile(formatTarget, os.O_RDWR, 0644)
	checkError(er)

	// write some text to file
	_, err = service_yml.WriteString(string(file))
	checkError(err)

	// save changes
	err = service_yml.Sync()
	checkError(err)

	service_yml.Close()

	//format long syntax ports
	service_yml, _ = os.Open(formatTarget)
	defer service_yml.Close()

	var lines []string
	scanner := bufio.NewScanner(service_yml)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	text := ""
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "cpu:") {
			lines[i] = formatCpu(lines[i])
		}
		if strings.Contains(lines[i], "env_vars:") {
			text += lines[i] + "\n"
			for i = i + 1; i < len(lines); i++ {
				if isEnv(lines[i]) {
					lines[i] = formatEnv_Vars(lines[i])
					text += lines[i] + "\n"
				} else {
					text += lines[i] + "\n"
					break
				}
			}
		} else {
			if lines[i] == "    - |-" {
				lines[i] = "    -"
			}
			text += lines[i] + "\n"
		}

	}

	//write the final service.yml
	service_yml, _ = os.Create(formatTarget)
	service_yml, er = os.OpenFile(formatTarget, os.O_RDWR, 0644)

	_, err = service_yml.WriteString(text)

	//rewrite the original docker-compose
	docker, _ := os.Create(filename)
	docker, er = os.OpenFile(filename, os.O_RDWR, 0644)

	err = docker.Sync()
	checkError(err)

	_, er = docker.WriteString(initialDockerText)
	docker.Close()

	err = os.Remove(auxFile)
	checkError(err)

	return true, err

}



func copyToServiceYML(d map[string]docker_Service, directlyTransformed bool) (map[string]common.Service, []string) {

	serviceYaml := Serviceyml{
		Services: make(map[string]common.Service),
		Dbs:      make([]string, 0),
	}
	var isDB bool
	var err error
	var dbs []string

	directlyTransformed = true
	for k, v := range d {
		var current_db string
		isDB = false

		if v.Image != "" {
			current_db, isDB = checkDB(v.Image)
		}
		if isDB {
			dbs = append(dbs, current_db)
		}
		if !isDB {
			var longSyntaxPorts []string
			longSyntaxPorts = v.Expose //expose and long syntax for ports dont work together..i think?
			if len(v.Ports.ShortSyntax) > 0 {
				for i := 0; i < len(v.Ports.ShortSyntax); i++ {
					longSyntaxPorts = append(longSyntaxPorts, v.Ports.ShortSyntax[i])
				}
			} else {
				longSyntaxPorts = []string{}
				for i := 0; i < len(v.Ports.Port); i++ {

					longSyntax := ""
					longSyntax = "target: " + v.Ports.Port[i].Target + "\n"

					if v.Ports.Port[i].Protocol == "udp" {
						longSyntax += "udp: " + v.Ports.Port[i].Published
						longSyntaxPorts = append(longSyntaxPorts, longSyntax)
					} else if v.Ports.Port[i].Protocol == "tcp" {
						reader := bufio.NewReader(os.Stdin)
						fmt.Printf("\nYou have chosen a TCP protocol for the port published at %s - should it be mapped as HTTP, HTTPS or TCP ? : ", v.Ports.Port[i].Published)
						var answer string
						answer, _ = reader.ReadString('\n')
						answer = strings.ToUpper(answer)
						if answer == "TCP\n" {
							longSyntax += "tcp: " + v.Ports.Port[i].Published
						} else if answer == "HTTP\n" {
							longSyntax += "http: " + v.Ports.Port[i].Published
						} else if answer == "HTTPS\n" {
							longSyntax += "http: " + v.Ports.Port[i].Published
						}
						longSyntaxPorts = append(longSyntaxPorts, longSyntax)
					}

				}
			}

			var gitPath string
			gitPath, err = common.GitRootDir("/")
			//checkError(err)
			if err != nil {

			}
			hasGit := common.HasGit(gitPath)

			var gitURL, gitBranch, buildRoot string
			if hasGit {
				//common.PrintlnWarning("HAS GIT here!")
				gitURL = common.RemoteGitUrl(gitPath)
				gitBranch = common.LocalGitBranch(gitPath)
				buildRoot, err = common.PathRelativeToGitRoot(gitPath)
			}

			var serviceYamlService common.Service
			serviceYamlService.GitRepo = gitURL
			serviceYamlService.GitBranch = gitBranch
			serviceYamlService.BuildRoot = buildRoot
			serviceYamlService.BuildCommand = v.Build_Command.Build_Command
			serviceYamlService.CommandSlice = v.Command.Command
			serviceYamlService.Image = v.Image
			serviceYamlService.Requires = v.Depends_on
			serviceYamlService.Volumes = v.Volumes.Volumes
			serviceYamlService.Stop_grace = v.Stop_grace_period
			serviceYamlService.Work_dir = v.Working_dir
			serviceYamlService.EnvVarsSlice = v.EnvVars
			serviceYamlService.Tags = v.Labels
			serviceYamlService.Dockerfile_path = v.Build_Command.Build.Dockerfile
			serviceYamlService.Privileged = v.Privileged
			serviceYamlService.Constraints = common.Constraints{
				Resources: common.Resources{
					Memory: v.Deploy.Resources.Limits.Memory,
					Cpu:    v.Deploy.Resources.Limits.Cpus,
				},
			}
			serviceYamlService.PortsShort = longSyntaxPorts

			if v.Env_file.Env_file != nil {
				var lines []string
				for i := 0; i < len(v.Env_file.Env_file); i++ {
					lines = readEnv_file(v.Env_file.Env_file[i])
					for j := 0; j < len(lines); j++ {
						serviceYamlService.EnvVarsSlice = append(serviceYamlService.EnvVarsSlice, lines[j])
					}
				}
			}

			//serviceYamlService.EnvVarsSlice = formatEnv_Vars(serviceYamlService.EnvVarsSlice)

			if serviceYamlService.Image != "" {
				serviceYamlService.GitRepo = ""
				serviceYamlService.GitBranch = ""
				serviceYamlService.BuildRoot = ""
			}
			// assign stuff via v, k is the service name
			serviceYaml.Services[k] = serviceYamlService

		}

	}
	return serviceYaml.Services, dbs
}
