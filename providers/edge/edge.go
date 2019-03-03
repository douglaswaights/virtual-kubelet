package edge

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"time"

	"github.com/virtual-kubelet/virtual-kubelet/manager"
	"github.com/virtual-kubelet/virtual-kubelet/providers"
	"go.opencensus.io/trace"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"

	log "github.com/Sirupsen/logrus"
)

var pathTooApplicationsDir string
var applicationsDirName = "applications"

// EdgeProvider a provider for running applications at the edge
type EdgeProvider struct {
	nodeName           string
	resourceManager    *manager.ResourceManager
	providerConfigPath string
	operatingSystem    string
	daemonEndpointPort int32
	pods               map[string]*v1.Pod
	startTime          time.Time
}

// NewEdgeProvider create a new provider
func NewEdgeProvider(providerConfigPath string, nodeName string, resourceManager *manager.ResourceManager, operatingSystem string, daemonEndPoint int32) (*EdgeProvider, error) {

	var filename = "C:/git/go/src/github.com/douglaswaights/edge-agent/logs/vk.log"
	f, _ := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	log.SetOutput(f)

	ex, _ := os.Executable()
	homeExeFullDir := filepath.Dir(ex)
	//pathTooApplicationsDir = homeExeFullDir + "/applications"
	pathTooApplicationsDir = filepath.FromSlash(homeExeFullDir + "/" + applicationsDirName)

	edge := EdgeProvider{
		nodeName:           nodeName,
		providerConfigPath: providerConfigPath,
		resourceManager:    resourceManager,
		operatingSystem:    operatingSystem,
		daemonEndpointPort: daemonEndPoint,
		pods:               make(map[string]*v1.Pod),
		startTime:          time.Now(),
	}
	return &edge, nil
}

// CreatePod takes a Kubernetes Pod and deploys it within the provider.
func (p *EdgeProvider) CreatePod(ctx context.Context, pod *v1.Pod) error {
	log.Printf("create pod requested " + pod.Name)
	err := createApp(pod)
	if err != nil {
		return err
	}
	p.pods[pod.Name] = pod
	return nil
}

// UpdatePod takes a Kubernetes Pod and updates it within the provider.
func (p *EdgeProvider) UpdatePod(ctx context.Context, pod *v1.Pod) error {
	log.Printf("update pod requested " + pod.Name)
	return nil
}

// DeletePod takes a Kubernetes Pod and deletes it from the provider.
func (p *EdgeProvider) DeletePod(ctx context.Context, pod *v1.Pod) error {
	log.Printf("delete pod requested " + pod.Name)
	processName := getProcessNameFromPodName(pod.Name)
	err := deleteApp(processName)
	if err != nil {
		return err
	}
	delete(p.pods, pod.Name)
	return nil
}

// GetPod retrieves a pod by name from the provider (can be cached).
func (p *EdgeProvider) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	log.Printf("get pod requested " + name)
	processName := getProcessNameFromPodName(name)
	running, err := isProcessRunning(processName)
	if err != nil {
		return nil, err
	}
	if !running {
		return nil, nil
	}
	pod := p.pods[name]
	return pod, nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *EdgeProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, tail int) (string, error) {
	log.Printf("get container logs requested " + podName + " containerName")
	return "", nil
}

// ExecInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *EdgeProvider) ExecInContainer(name string, uid types.UID, container string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize, timeout time.Duration) error {
	log.Printf("receive ExecInContainer %q\n", container)
	return nil
}

// GetPodStatus retrieves the status of a pod by name from the provider.
func (p *EdgeProvider) GetPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	log.Printf("Get pod status requested " + name)
	pod, err := p.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if pod == nil {
		return nil, nil
	}
	return &pod.Status, nil
}

// GetPods retrieves a list of all pods running on the provider (can be cached).
func (p *EdgeProvider) GetPods(context.Context) ([]*v1.Pod, error) {
	return nil, nil
}

// Capacity returns a resource list with the capacity constraints of the provider.
func (p *EdgeProvider) Capacity(context.Context) v1.ResourceList {
	return v1.ResourceList{
		"cpu":    resource.MustParse("20"),
		"memory": resource.MustParse("100Gi"),
		"pods":   resource.MustParse("20"),
	}
}

// NodeConditions returns a list of conditions (Ready, OutOfDisk, etc), which is
// polled periodically to update the node status within Kubernetes.
func (p *EdgeProvider) NodeConditions(context.Context) []v1.NodeCondition {
	return []v1.NodeCondition{
		{
			Type:               "Ready",
			Status:             v1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "Edge kubelet ready for action",
		},
	}
}

// NodeAddresses returns a list of addresses for the node status
// within Kubernetes.
func (p *EdgeProvider) NodeAddresses(context.Context) []v1.NodeAddress {
	return nil
}

// NodeDaemonEndpoints returns NodeDaemonEndpoints for the node status
// within Kubernetes.
func (p *EdgeProvider) NodeDaemonEndpoints(ctx context.Context) *v1.NodeDaemonEndpoints {
	ctx, span := trace.StartSpan(ctx, "NodeDaemonEndpoints")
	defer span.End()

	return &v1.NodeDaemonEndpoints{
		KubeletEndpoint: v1.DaemonEndpoint{
			Port: p.daemonEndpointPort,
		},
	}
}

// OperatingSystem returns the operating system the provider is for.
func (p *EdgeProvider) OperatingSystem() string {
	return providers.OperatingSystemLinux
}
