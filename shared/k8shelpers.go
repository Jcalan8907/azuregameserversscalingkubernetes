package shared

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	multiplayergameserverclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// NewPod returns a Kubernetes Pod struct
func NewPod(name string, port int32, setSessionsURL string) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"server": name},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "gameserver",
					Image: "docker.io/dgkanatsios/docker_openarena_k8s:latest",
					Ports: []core.ContainerPort{
						{
							Name:          "port1",
							Protocol:      core.ProtocolUDP,
							ContainerPort: port,
						},
					},
					Env: []core.EnvVar{
						{
							Name:  "OA_STARTMAP",
							Value: "dm4ish",
						},
						{
							Name:  "OA_PORT",
							Value: strconv.Itoa(int(port)),
						},
						{
							Name: "STORAGE_ACCOUNT_NAME",
							ValueFrom: &core.EnvVarSource{
								SecretKeyRef: &core.SecretKeySelector{
									LocalObjectReference: core.LocalObjectReference{
										Name: "openarena-storage-secret",
									},
									Key: "azurestorageaccountname",
								},
							},
						},
						{
							Name: "STORAGE_ACCOUNT_KEY",
							ValueFrom: &core.EnvVarSource{
								SecretKeyRef: &core.SecretKeySelector{
									LocalObjectReference: core.LocalObjectReference{
										Name: "openarena-storage-secret",
									},
									Key: "azurestorageaccountkey",
								},
							},
						},
						{
							Name:  "SERVER_NAME",
							Value: name,
						},
						{
							Name:  "SET_SESSIONS_URL",
							Value: setSessionsURL,
						},
					},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      "openarenavolume",
							MountPath: "/data",
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: "openarenavolume",
					VolumeSource: core.VolumeSource{
						AzureFile: &core.AzureFileVolumeSource{
							SecretName: "openarena-storage-secret",
							ShareName:  "openarenadata",
							ReadOnly:   false,
						},
					},
				},
			},
			HostNetwork:   true,
			DNSPolicy:     core.DNSClusterFirstWithHostNet, //https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
			RestartPolicy: core.RestartPolicyNever,
		},
	}
	return pod
}

// NewService returns a Kubernetes Service struct
func NewService(name string, port int32) *core.Service {
	service := &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{{
				Name:     "port1",
				Protocol: core.ProtocolUDP,
				Port:     port,
			}},
			Selector: map[string]string{"server": name},
			Type:     "LoadBalancer",
		},
	}
	return service
}

// GetClientSet returns a Kubernetes interface object that will allow us to give commands to the K8s API
func GetClientSet() (kubernetes.Interface, multiplayergameserverclientset.Interface) {
	if runInK8s := os.Getenv("RUN_IN_K8S"); runInK8s == "" || runInK8s == "true" {
		config, err := rest.InClusterConfig()

		if err != nil {
			fmt.Println(err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Println(err)
		}

		multiplayergameserverclientset, err := multiplayergameserverclientset.NewForConfig(config)
		if err != nil {
			log.Fatalf("getClusterConfig: %v", err)
		}

		return clientset, multiplayergameserverclientset
	}
	//else...
	clientset, multiplayergameserverclientset := GetClientOutOfCluster()
	return clientset, multiplayergameserverclientset

}

func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = "~/.kube/config"
		//kubeconfigPath = "C:\\users\\dgkanatsios\\.kube\\config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetClientOutOfCluster returns a k8s clientset to the request from outside of cluster
func GetClientOutOfCluster() (kubernetes.Interface, multiplayergameserverclientset.Interface) {
	config, err := buildOutOfClusterConfig()
	if err != nil {
		log.Fatalf("Can not get kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Fatalf("Can not create clientset for config: %v", err)
	}

	multiplayergameserverclientset, err := multiplayergameserverclientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("GetClientOutOfCluster: %v", err)
	}

	return clientset, multiplayergameserverclientset
}

//CreateKubeConfig authenticates to the local cluster
func CreateKubeConfig() *string {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	return kubeconfig
}
