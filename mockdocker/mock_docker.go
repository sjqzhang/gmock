package mockdocker

import (
	docker "github.com/fsouza/go-dockerclient"
	"log"
)

func getDockerClient() (*docker.Client, error) {

	return docker.NewClientFromEnv()
}

type MockService interface {
	InitContainer(handler func(opts *docker.CreateContainerOptions)) error
	Destroy()
}


type MockDockerService struct {
	client     *docker.Client
	containers []*docker.Container
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
	handler(&opts)
	//var pullOpts docker.PullImageOptions
	//pullOpts.Repository=opts.Config.Image
	//var auth docker.AuthConfiguration
	//m.client.PullImage(pullOpts,auth)
	return m.startService(opts)
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

}
