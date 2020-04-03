// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	types "github.com/vmware/hamlet/api/types/v1alpha1"

	log "github.com/sirupsen/logrus"
	"github.com/stevesloka/gimlet/server/internal/signals"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/golang/protobuf/proto"
	"github.com/vmware/hamlet/pkg/server"
	"github.com/vmware/hamlet/pkg/server/state"
	"github.com/vmware/hamlet/pkg/tls"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// emptyProvider is a sample state provider implementation that always returns a
// default empty set of resources.
type emptyProvider struct {
	state.StateProvider
	client client.Client
}

func (p *emptyProvider) GetState(msg string) ([]proto.Message, error) {
	entryLog := logger.WithName("GetState")

	svcs := &v1.ServiceList{}
	err := p.client.List(context.TODO(), svcs)
	if err != nil {
		entryLog.Error(err, "unable to get list of services")
	}

	var protoSvcs []proto.Message
	for _, svc := range svcs.Items {
		protoSvcs = append(protoSvcs, &types.FederatedService{
			Name:        fmt.Sprintf("%s-%s", svc.GetName(), svc.GetNamespace()),
			Description: "discovered Kubernetes service",
		})
	}

	return protoSvcs, nil
}

var logger = logf.Log.WithName("gimlet-controller")

// Start starts the server lifecycle.
func Start(rootCACerts []string, peerCert string, peerKey string, port uint32) {
	logf.SetLogger(zap.Logger(false))
	entryLog := logger.WithName("entrypoint")

	// Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	// Initialize the Hamlet server
	tlsConfig := tls.PrepareServerConfig(rootCACerts, peerCert, peerKey)
	s, err := server.NewServer(port, tlsConfig, &emptyProvider{client: mgr.GetClient()})
	if err != nil {
		log.WithField("err", err).Fatalln("Error occurred while creating the server instance")
	}

	s.Resources()

	// Setup a new controller to reconcile Services
	entryLog.Info("Setting up controller")
	c, err := controller.New("service-controller", mgr, controller.Options{
		Reconciler: &reconcileService{client: mgr.GetClient(), log: logger.WithName("reconciler"), server: s},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up individual controller")
		os.Exit(1)
	}

	// Watch ReplicaSets and enqueue ReplicaSet object key
	if err := c.Watch(&source.Kind{Type: &v1.Service{}}, &handler.EnqueueRequestForObject{}); err != nil {
		entryLog.Error(err, "unable to watch Services")
		os.Exit(1)
	}

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

	go func() {
		// Start the server.
		entryLog.Info("starting hamlet server")
		if err := s.Start(); err != nil {
			log.WithField("err", err).Errorln("Error occurred while starting the server")
		}
	}()

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}

}
