package quarksjob

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	qjv1a1 "code.cloudfoundry.org/quarks-job/pkg/kube/apis/quarksjob/v1alpha1"
	"code.cloudfoundry.org/quarks-job/pkg/kube/util/reference"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/names"
	vss "code.cloudfoundry.org/quarks-utils/pkg/versionedsecretstore"
)

// AddErrand creates a new QuarksJob controller to start errands, when their
// trigger strategy matches 'now' or 'once', or their configuration changed.
func AddErrand(ctx context.Context, config *config.Config, mgr manager.Manager) error {
	f := controllerutil.SetControllerReference
	ctx = ctxlog.NewContextWithRecorder(ctx, "errand-reconciler", mgr.GetEventRecorderFor("errand-recorder"))
	store := vss.NewVersionedSecretStore(mgr.GetClient())
	r := NewErrandReconciler(ctx, config, mgr, f, store)
	c, err := controller.New("errand-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: config.MaxQuarksJobWorkers,
	})
	if err != nil {
		return errors.Wrap(err, "Adding Errand controller to manager failed.")
	}

	// Trigger when
	//  * errand jobs are to be run (Spec.Run changes from `manual` to `now` or the job is created with `now`)
	//  * auto-errands with UpdateOnConfigChange == true have changed config references
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			qJob := e.Object.(*qjv1a1.QuarksJob)
			shouldProcessEvent := qJob.Spec.Trigger.Strategy == qjv1a1.TriggerNow || qJob.Spec.Trigger.Strategy == qjv1a1.TriggerOnce
			if shouldProcessEvent {
				ctxlog.NewPredicateEvent(qJob).Debug(
					ctx, e.Meta, qjv1a1.QuarksJobResourceName,
					fmt.Sprintf("Create predicate passed for '%s', existing quarksJob spec.Trigger.Strategy  matches the values 'now' or 'once'",
						e.Meta.GetName()),
				)
			}

			return shouldProcessEvent
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			o := e.ObjectOld.(*qjv1a1.QuarksJob)
			n := e.ObjectNew.(*qjv1a1.QuarksJob)

			enqueueForManualErrand := n.Spec.Trigger.Strategy == qjv1a1.TriggerNow && o.Spec.Trigger.Strategy == qjv1a1.TriggerManual

			// enqueuing for auto-errand when referenced secrets changed
			enqueueForConfigChange := n.IsAutoErrand() && n.Spec.UpdateOnConfigChange && hasConfigsChanged(o, n)

			shouldProcessEvent := enqueueForManualErrand || enqueueForConfigChange
			if shouldProcessEvent {
				ctxlog.NewPredicateEvent(o).Debug(
					ctx, e.MetaNew, qjv1a1.QuarksJobResourceName,
					fmt.Sprintf("Update predicate passed for '%s', a change in it´s referenced secrets have been detected",
						e.MetaNew.GetName()),
				)
			}

			return shouldProcessEvent
		},
	}

	err = c.Watch(&source.Kind{Type: &qjv1a1.QuarksJob{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return errors.Wrapf(err, "Watching Quarks jobs failed in Errand controller.")
	}

	// Watch config maps referenced by resource QuarksJob,
	// trigger auto errand if UpdateOnConfigChange=true and config data changed
	p = predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			o := e.ObjectOld.(*corev1.ConfigMap)
			n := e.ObjectNew.(*corev1.ConfigMap)

			return !reflect.DeepEqual(o.Data, n.Data)
		},
	}

	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			cm := a.Object.(*corev1.ConfigMap)

			if reference.SkipReconciles(ctx, mgr.GetClient(), cm) {
				return []reconcile.Request{}
			}

			reconciles, err := reference.GetReconciles(ctx, mgr.GetClient(), reference.ReconcileForQuarksJob, cm)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to calculate reconciles for config '%s': %v", cm.Name, err)
			}

			for _, reconciliation := range reconciles {
				ctxlog.NewMappingEvent(a.Object).Debug(ctx, reconciliation, "QuarksJob", a.Meta.GetName(), names.ConfigMap)
			}
			return reconciles
		}),
	}, p)
	if err != nil {
		return err
	}

	// Watch secrets referenced by resource QuarksJob
	// trigger auto errand if UpdateOnConfigChange=true and config data changed
	p = predicate.Funcs{
		// Only enqueuing versioned secret which has versionedSecret label
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*corev1.Secret)
			ok := vss.IsVersionedSecret(*o)
			// Skip initial version since it will trigger twice if the job has been created with
			// `Strategy: Once` and secrets are created afterwards
			if ok && vss.IsInitialVersion(*o) {
				return false
			}
			return ok
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		// React to updates on all referenced secrets
		UpdateFunc: func(e event.UpdateEvent) bool {
			o := e.ObjectOld.(*corev1.Secret)
			n := e.ObjectNew.(*corev1.Secret)

			return !reflect.DeepEqual(o.Data, n.Data)
		},
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			s := a.Object.(*corev1.Secret)

			if reference.SkipReconciles(ctx, mgr.GetClient(), s) {
				return []reconcile.Request{}
			}

			reconciles, err := reference.GetReconciles(ctx, mgr.GetClient(), reference.ReconcileForQuarksJob, s)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to calculate reconciles for secret '%s': %v", s.Name, err)
			}

			for _, reconciliation := range reconciles {
				ctxlog.NewMappingEvent(a.Object).Debug(ctx, reconciliation, "QuarksJob", a.Meta.GetName(), names.Secret)
			}

			return reconciles
		}),
	}, p)

	return err
}

// hasConfigsChanged return true if object's config references changed
func hasConfigsChanged(oldEJob, newEJob *qjv1a1.QuarksJob) bool {
	oldConfigMaps, oldSecrets := vss.GetConfigNamesFromSpec(oldEJob.Spec.Template.Spec.Template.Spec)
	newConfigMaps, newSecrets := vss.GetConfigNamesFromSpec(newEJob.Spec.Template.Spec.Template.Spec)

	if reflect.DeepEqual(oldConfigMaps, newConfigMaps) && reflect.DeepEqual(oldSecrets, newSecrets) {
		return false
	}

	// For versioned secret, we only enqueue changes for higher version of secrets
	for newSecret := range newSecrets {
		secretPrefix := vss.NamePrefix(newSecret)
		newVersion, err := vss.VersionFromName(newSecret)
		if err != nil {
			continue
		}

		if isLowerVersion(oldSecrets, secretPrefix, newVersion) {
			return false
		}
	}

	// other configs changes should be enqueued
	return true
}

func isLowerVersion(oldSecrets map[string]struct{}, secretPrefix string, newVersion int) bool {
	for oldSecret := range oldSecrets {
		if strings.HasPrefix(oldSecret, secretPrefix) {
			oldVersion, _ := vss.VersionFromName(oldSecret)

			if newVersion < oldVersion {
				return true
			}
		}
	}

	// if not found in old secrets, it's a new versioned secret
	return false
}
