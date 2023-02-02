package mockgrpc

import (
	"fmt"
	"github.com/sjqzhang/gmock/util"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

var logger = util.NewLogger("mockgrpc")

type MockGRPC struct {
	dirProtoc     string
	dirStub       string
	portAdmin     int
	portGrpc      int
	containerName string
}

// option for New MockGRPC
type OptionGRPC func(*MockGRPC)

// WithDirProtocs set the directory of protoc files
func WithDirProtocs(dir string) OptionGRPC {
	return func(m *MockGRPC) {
		m.dirProtoc = dir
	}
}

// WithDirStubs set the directory of stubs
func WithDirStubs(dir string) OptionGRPC {
	return func(m *MockGRPC) {
		m.dirStub = dir
	}
}

// WithPortAdmin set the port of admin
func WithPortAdmin(port int) OptionGRPC {
	return func(m *MockGRPC) {
		m.portAdmin = port
	}
}

// WithPortGrpc set the port of grpc
func WithPortGrpc(port int) OptionGRPC {
	return func(m *MockGRPC) {
		m.portGrpc = port
	}
}

// WithContainerName set the name of container
func WithContainerName(name string) OptionGRPC {
	return func(m *MockGRPC) {
		m.containerName = name
	}
}

// NewMockGRPC create a new MockGRPC
func NewMockGRPC(opts ...OptionGRPC) *MockGRPC {
	m := &MockGRPC{
		dirProtoc:     "protos",
		dirStub:       "stubs",
		portAdmin:     4771,
		portGrpc:      4770,
		containerName: "mock_grpc",
	}
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// findFiles find all files with the extension
func findFiles(root, ext string) []string {
	var a []string
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			a = append(a, s)
		}
		return nil
	})
	return a
}

// Start start the mock grpc server
func (m *MockGRPC) Start() error {

	// is absolute path
	if !filepath.IsAbs(m.dirProtoc) {
		s, err := filepath.Abs(m.dirProtoc)
		if err == nil {
			m.dirProtoc = s
		}
	}
	if !filepath.IsAbs(m.dirStub) {
		s, err := filepath.Abs(m.dirStub)
		if err == nil {
			m.dirStub = s
		}
	}
	files := findFiles(m.dirProtoc, ".proto")
	// quotes each file with '
	for i, f := range files {

		files[i] = fmt.Sprintf("'/proto/%s'", strings.TrimPrefix(f, m.dirProtoc+"/"))
	}
	cmd := fmt.Sprintf("docker run --rm -p %d:4771 -p %d:4770 -v %s:/proto -v %s:/stub --name %s  sjqzhang/gripmock --stub /stub %s ",
		m.portAdmin, m.portGrpc, m.dirProtoc, m.dirStub, m.containerName, strings.Join(files, " "))

	logger.Log(cmd)
	var err error
	go func() {
		ret, exitCode := util.Exec(cmd)
		if exitCode != 0 {
			logger.Log(fmt.Sprintf("start mock grpc server failed: %s", ret))
		}
		err = fmt.Errorf("start mock grpc server failed: %s", ret)
	}()
	<-time.After(3 * time.Second)
	return err
}

// Stop stop the mock grpc server
func (m *MockGRPC) Stop() error {
	ret, exitCode := util.Exec(fmt.Sprintf("docker stop  %s", m.containerName))
	if exitCode != 0 {
		return fmt.Errorf("stop mock grpc server failed: %s", ret)
	}
	return nil
}


// GetAddr get the address of mock grpc server
func (m *MockGRPC) GetAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", m.portGrpc)
}