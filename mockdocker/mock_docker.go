package mockdocker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func getDockerClient() (*docker.Client, error) {

	return docker.NewClientFromEnv()
}

type Logger struct {
	tag string
	log *log.Logger
}

func NewLogger(tag string) *Logger {
	return &Logger{tag: tag, log: log.New(os.Stdout, fmt.Sprintf("[%v] ", tag), log.LstdFlags)}
}

func (l *Logger) Log(msg interface{}) {
	l.log.Println("\u001B[32m" + fmt.Sprintf("%v", msg)  + "\u001B[0m")
}
func (l *Logger) Warn(msg interface{}) {
	l.log.Println("\u001B[33m" + fmt.Sprintf("%v", msg)  + "\u001B[0m")
}
func (l *Logger) Error(msg interface{}) {

	l.log.Println("\u001B[31m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *Logger) Panic(msg interface{}) {
	panic("\u001B[31m" + fmt.Sprintf("%v", msg)  + "\u001B[0m")
}

var logger = NewLogger("gmock.mockdocker")


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

//InitContainer 通过容器参数实例化容器
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

// InitContainerWithCmd 通过命令行参数初始化容器
func (m *MockDockerService) InitContainerWithCmd(handler func(cmd *string)) error {
	var cmd string
	handler(&cmd)
	return m.startServiceWithCmd(cmd)
}

func (m *MockDockerService) startServiceWithCmd(cmdStr string) error {
	if strings.Index(cmdStr, " -d ") == -1 {
		return errors.New("(warning)Container must be run in background, run with -d option")
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
	defer func(stdout io.ReadCloser, stderr io.ReadCloser) {
		if stderr != nil {
			_ = stderr.Close()
		}
		if stdout != nil {
			_ = stdout.Close()
		}
	}(stdout, stderr)
	if err = cmdProcess.Start(); err != nil {
		return err
	}
	errBytes, _ := ioutil.ReadAll(stderr)
	if len(errBytes) > 0 {
		logger.Log(string(errBytes))
	}
	opBytes, err := ioutil.ReadAll(stdout)
	if len(opBytes) > 64 {
		id := opBytes[0:63]
		if regexp.MustCompile("[a-z0-9]").Match(id) {
			m.containerIDs = append(m.containerIDs, string(id))
		}
	} else {
		logger.Log(string(opBytes))
	}
	return nil
}

//WaitForReady 通过命令检测容器是否准备完成
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
		err := process.Run()
		if err != nil {
			logger.Error(err)
		}
		logger.Log(string(buf.Bytes()))
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

// Destroy 销毁容器
func (m *MockDockerService) Destroy() {
	for _, container := range m.containers {
		id := container.ID
		err := m.client.StopContainer(id, 15)
		if err != nil {
			logger.Error(err)
		}
		var removeContainer docker.RemoveContainerOptions
		removeContainer.ID = id
		removeContainer.Force = true
		err2 := m.client.RemoveContainer(removeContainer)
		if err2 != nil {
			logger.Error(err)
		}
	}

	for _, containerid := range m.containerIDs {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*15)
		//defer cannel()
		cmd := exec.CommandContext(ctx, "docker", "stop", containerid)
		err := cmd.Run()
		if err != nil {
			logger.Error(err)
		}
		cmd = exec.Command("docker", "rm", containerid)
		err = cmd.Run()
		if err != nil {
			logger.Error(err)
		}
	}

}
