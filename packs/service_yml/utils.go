package service_yml

import (
	"fmt"
	"os"
	"strings"
	"strconv"
	"gopkg.in/yaml.v2"
)

func handleEnvVarsFormat(file []byte) string {
	finalFormat := ""

	lines := strings.Split(string(file), "\n")

	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "_env") {
			for j := 0; j < len(lines[i])-4; j++ {
				if lines[i][j] == '_' && lines[i][j+1] == 'e' && lines[i][j+2] == 'n' && lines[i][j+3] == 'v' {
					lines[i] = lines[i][:j] + "$" + lines[i][j+4:]
				}
			}
		}
		finalFormat = finalFormat + lines[i] + "\n"
	}

	return finalFormat
}

func handleVolumes(serviceVolumes []string) []VolumeMounts {
	var kubeVolumes []VolumeMounts

	for _, volume := range serviceVolumes {
		name := ""
		mountPath := ""
		var i int
		var readOnly bool
		if volume[0] == '"' {
			i = 1
		} else {
			i = 0
		}
		for ; i < len(volume); i++ {
			if volume[i] == ':' {
				break
			} else {
				name = string(append([]byte(name), volume[i]))
			}
		}

		for i = i + 1; i < len(volume); i++ {
			if volume[i] == ':' || volume[i] == '"' || volume[i] == '\n' {
				break
			} else {
				mountPath = string(append([]byte(mountPath), volume[i]))
			}
		}
		if i < len(volume)-2 {
			if volume[i] == ':' && volume[i+1] == 'r' && volume[i+2] == 'o' {
				readOnly = true
			}
		}
		kubeVolume := VolumeMounts{
			Name:      name,
			MountPath: mountPath,
			ReadOnly:  readOnly,
		}
		kubeVolumes = append(kubeVolumes, kubeVolume)
	}

	return kubeVolumes
}

func getKeysValues(env_vars map[string]string) ([]interface{}, []interface{}) {
	keys := []interface{}{}
	values := []interface{}{}
	for k, v := range env_vars {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}

func generatePortsFromShortSyntax(shortSyntax string, clusterPorts []KubesPorts, nodePorts []KubesPorts) ([]KubesPorts, []KubesPorts, []KubesPorts) {
	var dPorts []KubesPorts
	container := ""
	http := ""
	https := ""
	var i int

	for i = 0; i < len(shortSyntax); i++ {
		if shortSyntax[i] == ':' {
			break
		} else {
			container = string(append([]byte(container), shortSyntax[i]))
		}
	}

	for i = i + 1; i < len(shortSyntax); i++ {
		if shortSyntax[i] == ':' {
			break
		} else {
			http = string(append([]byte(http), shortSyntax[i]))
		}
	}

	for i = i + 1; i < len(shortSyntax); i++ {
		if shortSyntax[i] == '"' || shortSyntax[i] == '\n' {
			break
		} else {
			https = string(append([]byte(https), shortSyntax[i]))
		}
	}

	if http == "" && https == "" {
		//crate new node of type ClusterIP
		clusterPorts = appendNewPort(clusterPorts, container+"-expose", container, container, "", "")
		dPorts = appendNewPort(dPorts, container+"-expose", "", "", "", container)
	}
	if http != "" {
		nodePorts = appendNewPort(nodePorts, container+"-http", container, http, "tcp", "")
		dPorts = appendNewPort(dPorts, container+"-http", "", "", "tcp", container)
		//create new port with name "http" of type NodePort
	}
	if https != "" {
		//create new port with name "https of type NodePort
		nodePorts = appendNewPort(nodePorts, container+"-https", container, https, "tcp", "")
		dPorts = appendNewPort(dPorts, container+"-https", "", "", "tcp", container)
	}

	return dPorts, clusterPorts, nodePorts
}

func generatePortsFromLongSyntax(longSyntax ServicePort, clusterPorts []KubesPorts, nodePorts []KubesPorts) ([]KubesPorts, []KubesPorts, []KubesPorts) {
	var dPorts []KubesPorts

	if longSyntax.Tcp == "" && longSyntax.Https == "" && longSyntax.Https == "" && longSyntax.Udp == "" {
		//type "ClusterIP"
		clusterPorts = appendNewPort(clusterPorts, longSyntax.Container+"-expose", longSyntax.Container, longSyntax.Container, "", "")
		dPorts = appendNewPort(dPorts, longSyntax.Container+"-expose", "", "", "", longSyntax.Container)
	}
	if longSyntax.Udp != "" {
		nodePorts = appendNewPort(nodePorts, longSyntax.Container+"-udp", longSyntax.Container, longSyntax.Udp, "udp", "")
		dPorts = appendNewPort(dPorts, longSyntax.Container+"-udp", "", "", "udp", longSyntax.Container)
	}
	if longSyntax.Tcp != "" {
		nodePorts = appendNewPort(nodePorts, longSyntax.Container+"-tcp", longSyntax.Container, longSyntax.Tcp, "tcp", "")
		dPorts = appendNewPort(dPorts, longSyntax.Container+"-tcp", "", "", "tcp", longSyntax.Container)
	}
	if longSyntax.Http != "" {
		nodePorts = appendNewPort(nodePorts, longSyntax.Container+"-http", longSyntax.Container, longSyntax.Http, "tcp", "")
		dPorts = appendNewPort(dPorts, longSyntax.Container+"-http", "", "", "tcp", longSyntax.Container)
	}
	if longSyntax.Https != "" {
		nodePorts = appendNewPort(nodePorts, longSyntax.Container+"-https", longSyntax.Container, longSyntax.Https, "tcp", "")
		dPorts = appendNewPort(dPorts, longSyntax.Container+"-https", "", "", "tcp", longSyntax.Container)
	}

	return dPorts, clusterPorts, nodePorts
}

func appendNewPort(ports []KubesPorts, name string, port string, targetPort string, protocol string, containerPort string) []KubesPorts {
	ports = append(ports, KubesPorts{
		Name:          name,
		Port:          port,
		TargetPort:    targetPort,
		Protocol:      protocol,
		ContainerPort: containerPort,
	})
	return ports
}

func handlePorts(serviceName string, serviceSpecs ServiceYMLService) ([]KubesPorts, []KubesService) {
	services := []KubesService{}
	var dPorts, cPorts, nPorts, clusterPorts, nodePorts, deployPorts []KubesPorts
	for _, v := range serviceSpecs.Ports {
		switch v.(type) {
		case string:
			shortSyntax := v.(string)
			dPorts, cPorts, nPorts = generatePortsFromShortSyntax(shortSyntax, clusterPorts, nodePorts)
		case int:
			shortSyntax := strconv.Itoa(v.(int))
			dPorts, cPorts, nPorts = generatePortsFromShortSyntax(shortSyntax, clusterPorts, nodePorts)
		case map[interface{}]interface{}:
			var longSyntaxPort ServicePort
			temp, er := yaml.Marshal(v)
			CheckError(er)
			er = yaml.Unmarshal(temp, &longSyntaxPort)
			CheckError(er)
			dPorts, cPorts, nPorts = generatePortsFromLongSyntax(longSyntaxPort, clusterPorts, nodePorts)
		}

		for _, port := range dPorts {
			deployPorts = append(deployPorts, port)
		}
		for _, port := range cPorts {
			clusterPorts = append(clusterPorts, port)
		}
		for _, port := range nPorts {
			nodePorts = append(nodePorts, port)
		}
	}

	//generate services with the specific type required by the found nodes
	if len(clusterPorts) > 0 {
		clusterService := generateService("ClusterIP", serviceSpecs, serviceName, clusterPorts)
		services = append(services, clusterService)
	}
	if len(nodePorts) > 0 {
		nodeService := generateService("NodePorts", serviceSpecs, serviceName, nodePorts)
		services = append(services, nodeService)
	}

	return deployPorts, services
}

func generateService(serviceType string, serviceSpecs ServiceYMLService, serviceName string, ports []KubesPorts) KubesService {
	service := KubesService{}
	service = KubesService{ApiVersion: "extensions/v1beta1",
		Kind:                      "Service",
		Metadata: Metadata{
			Name:   serviceName + "-svc",
			Labels: serviceSpecs.Tags,
		},
		Spec: Spec{
			Type:  serviceType,
			Ports: ports,
		},
	}

	return service
}

func CheckError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
