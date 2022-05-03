package mockdocker

import (
	"bytes"
	"context"
	"errors"
	docker "github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func getDockerClient() (*docker.Client, error) {

	return docker.NewClientFromEnv()
}

type MockService interface {
	InitContainer(handler func(opts *docker.CreateContainerOptions)) error
	Destroy()
}

type MockDockerService struct {
	client       *docker.Client
	containers   []*docker.Container
	containerIDs []string
}

func NewMockDockerService() *MockDockerService {
	client, err := getDockerClient()
	if err != nil {
		panic(err)
	}
	return &MockDockerService{
		client: client,
		//containers: make([]*docker.Container, 10),
	}
}
func (m *MockDockerService) InitContainer(handler func(opts *docker.CreateContainerOptions)) error {
	var opts docker.CreateContainerOptions
	var config docker.Config
	var hostConfig docker.HostConfig
	opts.Config = &config
	opts.HostConfig = &hostConfig
	portBinder := make(map[docker.Port][]docker.PortBinding)
	opts.HostConfig.PortBindings = portBinder
	handler(&opts)
	return m.startService(opts)
}

func (m *MockDockerService) InitContainerWithCmd(handler func(cmd *string)) error {
	var cmd string
	handler(&cmd)
	return m.startServiceWithCmd(cmd)
}

func (m *MockDockerService) startServiceWithCmd(cmdStr string) error {
	if strings.Index(cmdStr, " -d ") == -1 {
		return errors.New("(warning)container must be run in background, run with -d option!!!!")
	}
	exp := regexp.MustCompile(`\s+`)
	cmdStr = strings.TrimSpace(cmdStr)
	cmds := exp.Split(cmdStr, -1)
	cmdProcess := exec.Command(cmds[0], cmds[1:]...)
	stdout, err := cmdProcess.StdoutPipe()
	stderr, err := cmdProcess.StderrPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	if err = cmdProcess.Start(); err != nil {
		return err
	}
	errBytes, _ := ioutil.ReadAll(stderr)
	if len(errBytes) > 0 {
		log.Println(string(errBytes))
	}
	opBytes, err := ioutil.ReadAll(stdout)
	if len(opBytes) > 64 {
		id := opBytes[0:63]
		if regexp.MustCompile("[a-z0-9]").Match(id) {
			m.containerIDs = append(m.containerIDs, string(id))
		}
	} else {
		log.Println(string(opBytes))
	}
	return nil
}
func (m *MockDockerService) WaitForReady(checkReadyCommand string, timeout time.Duration) bool {
	checkReadyCommand = strings.TrimSpace(checkReadyCommand)
	ticker := time.NewTicker(time.Second)
	exp := regexp.MustCompile(`\s+`)
	cmds := exp.Split(checkReadyCommand, -1)
	start := time.Now()
	for {
		<-ticker.C
		if time.Now().Sub(start) > timeout {
			return false
		}
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		process := exec.CommandContext(ctx, cmds[0], cmds[1:]...)
		var buf bytes.Buffer
		process.Stdout = &buf
		process.Stderr = &buf
		process.Run()
		log.Println(string(buf.Bytes()))
		if process.ProcessState.Sys().(syscall.WaitStatus).ExitStatus() == 0 {
			return true
		}
	}
}
func (m *MockDockerService) startService(config docker.CreateContainerOptions) error {

	container, err := m.client.CreateContainer(config)
	if err != nil {
		return err
	}
	err = m.client.StartContainer(container.ID, config.HostConfig)
	if err != nil {
		return err
	}
	m.containers = append(m.containers, container)
	//m.container = container
	return nil

}

func (m *MockDockerService) Destroy() {
	for _, container := range m.containers {
		id := container.ID
		err := m.client.StopContainer(id, 15)
		if err != nil {
			log.Println(err)
		}
		var removeContainer docker.RemoveContainerOptions
		removeContainer.ID = id
		removeContainer.Force = true
		err2 := m.client.RemoveContainer(removeContainer)
		if err2 != nil {
			log.Println(err)
		}
	}

	for _, containerid := range m.containerIDs {
		ctx := context.Background()
		ctx, cannel := context.WithTimeout(ctx, time.Second*15)
		defer cannel()
		cmd := exec.CommandContext(ctx, "docker", "stop", containerid)
		cmd.Run()
		cmd = exec.Command("docker", "rm", containerid)
		cmd.Run()
	}

}
