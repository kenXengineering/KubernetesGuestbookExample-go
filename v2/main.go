package main

import (
	"flag"
	"log"
	"os"

	"github.com/pkg/errors"
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/kubernetes/typed/extensions/v1beta1"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/v1"
	v1beta1Ext "k8s.io/client-go/1.4/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.4/pkg/util/intstr"
	"k8s.io/client-go/1.4/tools/clientcmd"
)

var (
	kubeconfig                   = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	namespace                    = "default"
	apiVersionExtensions         = "extensions/v1beta1"
	typeMetaExtensionsDeployment = unversioned.TypeMeta{APIVersion: apiVersionExtensions, Kind: "Deployment"}
	serviceTypeMeata             = unversioned.TypeMeta{Kind: "Service", APIVersion: "1"}
)

// KubeClient host the Kubernetes Clientset object used to communicate with Kubernetes
type KubeClient struct {
	ExtensionsClient *v1beta1.ExtensionsClient
	ClientSet        *kubernetes.Clientset
}

type Deployment struct {
	Name      string
	Replicas  int32
	Labels    map[string]string
	Container Container
}

type Container struct {
	Name          string
	Image         string
	PortName      string
	ContainerPort int32
	HostPort      int32
}

// Service host the speicifation for a Service
type Service struct {
	Labels      map[string]string
	Name        string
	ServicePort int32
	TargetPort  intstr.IntOrString
	Type        v1.ServiceType
}

func main() {
	flag.Parse()
	kubeClient, err := GetNewKubeClient(kubeconfig)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	kubeClient.CreateGuestbook()
}

// GetNewKubeClient returns a new KubeClient with a Kubernetes Client Set and Extensions Client
func GetNewKubeClient(configPath *string) (*KubeClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", *configPath)
	if err != nil {
		return nil, errors.Wrap(err, "Could not read config file")
	}
	log.Println("Getting Client")
	extensionsClient, err := v1beta1.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Kubernetes Client")
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Kubernetes Client")
	}
	kc := &KubeClient{
		ExtensionsClient: extensionsClient,
		ClientSet:        clientSet,
	}
	return kc, nil
}

func (kc *KubeClient) CreateGuestbook() {
	log.Println("Creating Redis Master Deployment")
	redisMasterDep := &Deployment{
		Name:     "redis-master",
		Replicas: int32(1),
		Labels: map[string]string{
			"app":  "redis",
			"role": "master",
		},
		Container: Container{
			Name:          "redis-master",
			Image:         "redis:2.8.23",
			PortName:      "redis-server",
			ContainerPort: int32(6379),
		},
	}

	err := kc.createDeployment(redisMasterDep)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating Redis Master Service")
	redisMasterSvc := &Service{
		Name: "redis-master",
		Labels: map[string]string{
			"app":  "redis",
			"role": "master",
		},
		ServicePort: int32(6379),
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: "redis-server",
		},
	}
	err = kc.createService(redisMasterSvc)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating Redis Slave Deployment")
	redisSlaveDep := &Deployment{
		Name:     "redis-slave",
		Replicas: int32(2),
		Labels: map[string]string{
			"app":  "redis",
			"role": "slave",
		},
		Container: Container{
			Name:          "redis-slave",
			Image:         "kubernetes/redis-slave:v2",
			PortName:      "redis-server",
			ContainerPort: int32(6379),
		},
	}

	err = kc.createDeployment(redisSlaveDep)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating Redis Slave Service")
	redisSlaveSvc := &Service{
		Name: "redis-slave",
		Labels: map[string]string{
			"app":  "redis",
			"role": "slave",
		},
		ServicePort: int32(6379),
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: "redis-server",
		},
	}
	err = kc.createService(redisSlaveSvc)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating Guestbook Deployment")
	guestbookDep := &Deployment{
		Name: "guestbook",
		Labels: map[string]string{
			"app": "guestbook",
		},
		Replicas: int32(3),
		Container: Container{
			Name:          "guestbook",
			Image:         "gcr.io/google_containers/guestbook:v3",
			PortName:      "http-server",
			ContainerPort: int32(3000),
		},
	}
	err = kc.createDeployment(guestbookDep)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating Guestbook Service")
	guestbookService := &Service{
		Name: "guestbook",
		Labels: map[string]string{
			"app": "guestbook",
		},
		ServicePort: int32(3000),
		TargetPort: intstr.IntOrString{
			StrVal: "http-server",
			Type:   intstr.String,
		},
		Type: v1.ServiceTypeLoadBalancer,
	}

	err = kc.createService(guestbookService)
	if err != nil {
		log.Println(err)
	}

}

func (kc *KubeClient) createDeployment(dep *Deployment) error {
	deployment := &v1beta1Ext.Deployment{
		TypeMeta: typeMetaExtensionsDeployment,
		ObjectMeta: v1.ObjectMeta{
			Name: dep.Name,
		},
		Spec: v1beta1Ext.DeploymentSpec{
			Replicas: &dep.Replicas,
			Selector: &v1beta1Ext.LabelSelector{
				MatchLabels: dep.Labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: dep.Labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  dep.Container.Name,
							Image: dep.Container.Image,
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          dep.Container.Name,
									ContainerPort: dep.Container.ContainerPort,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := kc.ExtensionsClient.Deployments(namespace).Create(deployment)
	if err != nil {
		return errors.Wrap(err, "Error creating deployment")
	}
	return nil
}

func (kc *KubeClient) createService(service *Service) error {
	svc := &v1.Service{
		TypeMeta: serviceTypeMeata,
		ObjectMeta: v1.ObjectMeta{
			Name:   service.Name,
			Labels: service.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Port:       service.ServicePort,
					TargetPort: service.TargetPort,
				},
			},
			Selector: service.Labels,
			Type:     service.Type,
		},
	}

	_, err := kc.ClientSet.Services(namespace).Create(svc)
	if err != nil {
		return errors.Wrap(err, "Error creating service")
	}
	return nil
}
