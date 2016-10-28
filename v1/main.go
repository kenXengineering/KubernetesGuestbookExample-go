package main

import (
	"flag"
	"log"

	"github.com/pkg/errors"

	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api/unversioned"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/util/intstr"
	"k8s.io/client-go/1.4/tools/clientcmd"
)

var (
	kubeconfig      = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	namespace       = "default"
	repContTypeMeta = unversioned.TypeMeta{
		Kind:       "RepliationController",
		APIVersion: "1",
	}
	serviceTypeMeata = unversioned.TypeMeta{
		Kind:       "Service",
		APIVersion: "1",
	}
)

// KubeClient host the Kubernetes Clientset object used to communicate with Kubernetes
type KubeClient struct {
	clientset *kubernetes.Clientset
}

// ReplicationController holds the specification for a Replication Controller
type ReplicationController struct {
	Labels     map[string]string
	Name       string
	Image      string
	PortName   string
	PortNumber int32
	Replicas   int32
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
	log.Println("Read config")
	clientset, err := getClient(kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	kubeClient := &KubeClient{
		clientset: clientset,
	}
	log.Println("Creating Guestbook")
	kubeClient.CreateGuestBook()
	log.Println("Created Guestbook")
}

func getClient(configPath *string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", *configPath)
	if err != nil {
		return nil, errors.Wrap(err, "Could not read config file")
	}
	log.Println("Getting Client")
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Kubernetes Client")
	}
	return clientset, nil
}

func (kc *KubeClient) createReplicationController(repCont *ReplicationController) error {
	rc := &v1.ReplicationController{
		TypeMeta: repContTypeMeta,
		ObjectMeta: v1.ObjectMeta{
			Name:   repCont.Name,
			Labels: repCont.Labels,
		},
		Spec: v1.ReplicationControllerSpec{
			Replicas: &repCont.Replicas,
			Selector: repCont.Labels,
			Template: &v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: repCont.Labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  repCont.Name,
							Image: repCont.Image,
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          repCont.PortName,
									ContainerPort: repCont.PortNumber,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := kc.clientset.ReplicationControllers(namespace).Create(rc)
	if err != nil {
		return errors.Wrap(err, "Error creating replication controller")
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

	_, err := kc.clientset.Services(namespace).Create(svc)
	if err != nil {
		return errors.Wrap(err, "Error creating service")
	}
	return nil
}

// CreateGuestBook will create the guestbook app
func (kc *KubeClient) CreateGuestBook() {

	log.Println("Creating redis-master Replication Controller")

	redisRepCont := &ReplicationController{
		Name: "redis-master",
		Labels: map[string]string{
			"app":  "redis",
			"role": "master",
		},
		Image:      "redis:2.8.23",
		PortName:   "redis-server",
		PortNumber: int32(6379),
		Replicas:   int32(1),
	}

	err := kc.createReplicationController(redisRepCont)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating redis-master Service")

	redisService := &Service{
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

	err = kc.createService(redisService)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating redis-slave Replication Controller")

	redisSlaveRepCont := &ReplicationController{
		Name: "redis-slave",
		Labels: map[string]string{
			"app":  "redis",
			"role": "slave",
		},
		Image:      "kubernetes/redis-slave:v2",
		PortName:   "redis-server",
		PortNumber: int32(6379),
		Replicas:   int32(3),
	}

	err = kc.createReplicationController(redisSlaveRepCont)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating redis-slave Service")

	redisSlaveService := &Service{
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

	err = kc.createService(redisSlaveService)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating guestbook Replication Controller")

	guestBookRepCont := &ReplicationController{
		Name: "guestbook",
		Labels: map[string]string{
			"app": "guestbook",
		},
		Image:      "gcr.io/google_containers/guestbook:v3",
		PortName:   "http-server",
		PortNumber: int32(3000),
		Replicas:   int32(3),
	}

	err = kc.createReplicationController(guestBookRepCont)
	if err != nil {
		log.Println(err)
	}

	log.Println("Creating guestbook Service")

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
