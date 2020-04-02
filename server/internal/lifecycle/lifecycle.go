// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	types "github.com/vmware/hamlet/api/types/v1alpha1"
	"github.com/vmware/hamlet/pkg/server"
	"github.com/vmware/hamlet/pkg/server/state"
	"github.com/vmware/hamlet/pkg/tls"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/clientcmd"
)

// emptyProvider is a sample state provider implementation that always returns a
// default empty set of resources.
type emptyProvider struct {
	state.StateProvider
}

func (p *emptyProvider) GetState(msg string) ([]proto.Message, error) {
	fmt.Println("--- GetState! ", msg)

	svc := &types.FederatedService{
		Name: "new client",
	}

	return []proto.Message{svc}, nil
}

// Start starts the server lifecycle.
func Start(rootCACerts []string, peerCert string, peerKey string, port uint32, inCluster bool, kubeconfig string) {
	// Initialize the server.
	tlsConfig := tls.PrepareServerConfig(rootCACerts, peerCert, peerKey)
	s, err := server.NewServer(port, tlsConfig, &emptyProvider{})
	if err != nil {
		log.WithField("err", err).Fatalln("Error occurred while creating the server instance")
	}

	s.Resources()

	// Init Kubernetes Client
	config, err := newRestConfig(kubeconfig, inCluster)
	if err != nil {
		log.Error(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(client, 0)

	// create the service watcher
	// obtain references to shared index informers for the services types.
	serviceInformer := kubeInformerFactory.Core().V1().Services()

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Pod than the version which was responsible for triggering the update.
	// Set up an event handler for when Service resources change.
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			notifyResourceChangesAdd(obj.(*v1.Service), s)
		},
		UpdateFunc: func(old, new interface{}) {
			notifyResourceChangesUpdate(new.(*v1.Service), s)
		},
		DeleteFunc: func(obj interface{}) {
			notifyResourceChangesDelete(obj.(*v1.Service), s)
		},
	})

	// Setup the shutdown goroutine.
	stopCh := make(chan struct{})
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChannel
		close(stopCh)
		if err := s.Stop(); err != nil {
			log.WithField("err", err).Errorln("Error occurred while starting the server")
		}
		os.Exit(0)
	}()

	go kubeInformerFactory.Start(stopCh)

	// Start the server.
	if err := s.Start(); err != nil {
		log.WithField("err", err).Errorln("Error occurred while starting the server")
	}

	// Wait forever
	select {}
}

func newRestConfig(kubeconfig string, inCluster bool) (*rest.Config, error) {
	if kubeconfig != "" && !inCluster {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// notifyResourceChangesAdd notifies consumers about the changes in resources.
func notifyResourceChangesAdd(service *v1.Service, s server.Server) error {
	// Create a new service.
	svc := &types.FederatedService{
		Name: service.GetName(),
		Id:   fmt.Sprintf("%s.%s.foo.com", service.GetName(), service.GetNamespace()),
	}
	if err := s.Resources().Create(svc); err != nil {
		log.WithField("svc", svc).Errorln("Error occurred while creating service")
		return err
	}
	log.WithField("svc", svc).Infoln("Successfully created a service")

	return nil
}

// notifyResourceChangesAdd notifies consumers about the changes in resources.
func notifyResourceChangesUpdate(service *v1.Service, s server.Server) error {
	// Create a new service.
	svc := &types.FederatedService{
		Name: service.GetName(),
		Id:   fmt.Sprintf("%s.%s.foo.com", service.GetName(), service.GetNamespace()),
	}
	if err := s.Resources().Update(svc); err != nil {
		log.WithField("svc", svc).Errorln("Error occurred while updating service")
		return err
	}
	log.WithField("svc", svc).Infoln("Successfully updated a service")

	return nil
}

// notifyResourceChangesAdd notifies consumers about the changes in resources.
func notifyResourceChangesDelete(service *v1.Service, s server.Server) error {
	// Create a new service.
	svc := &types.FederatedService{
		Name: service.GetName(),
		Id:   fmt.Sprintf("%s.%s.foo.com", service.GetName(), service.GetNamespace()),
	}
	if err := s.Resources().Delete(svc); err != nil {
		log.WithField("svc", svc).Errorln("Error occurred while deleting service")
		return err
	}
	log.WithField("svc", svc).Infoln("Successfully deleted a service")

	return nil
}
