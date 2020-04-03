package lifecycle

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	types "github.com/vmware/hamlet/api/types/v1alpha1"
	"github.com/vmware/hamlet/pkg/server"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileService reconciles ReplicaSets
type reconcileService struct {
	// client can be used to retrieve objects from the APIServer.
	client client.Client
	server server.Server

	log logr.Logger
}

// Implement reconcile.Reconciler so the controller can reconcile objects
var _ reconcile.Reconciler = &reconcileService{}

func (r *reconcileService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// set up a convenient log object so we don't have to type request over and over again
	log := r.log.WithValues("request", request)

	// Fetch the ReplicaSet from the cache
	svc := &v1.Service{}
	err := r.client.Get(context.TODO(), request.NamespacedName, svc)
	if errors.IsNotFound(err) {
		log.Error(nil, "Could not find Service")

		notifyResourceChangesDelete(svc, r.server)

		return reconcile.Result{}, nil
	}

	if err != nil {
		log.Error(err, "Could not fetch Service")
		return reconcile.Result{}, err
	}

	// Print the ReplicaSet
	log.Info("Reconciling Service", "container name", svc.GetName())

	notifyResourceChangesUpdate(svc, r.server)

	return reconcile.Result{}, nil
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
