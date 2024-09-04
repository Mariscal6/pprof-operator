/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	collectorv1 "github.com/Mariscal6/pprof-operator/api/v1"
	"github.com/Mariscal6/pprof-operator/internal/collector"
	"github.com/robfig/cron"
)

const baseDir = "pprof-operator"
const defaultProfileName = "default.pgo"

// PprofCollectorReconciler reconciles a PprofCollector object
type PprofCollectorReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	PprofCollector collector.ProfileCollector
}

// +kubebuilder:rbac:groups=collector.profiling.operators,resources=pprofcollectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=collector.profiling.operators,resources=pprofcollectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=collector.profiling.operators,resources=pprofcollectors/finalizers,verbs=update
func (r *PprofCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	var collector collectorv1.PprofCollector
	if err := r.Get(ctx, req.NamespacedName, &collector); err != nil {
		log.Error(err, "unable to fetch PprofCollector")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(req.Namespace),
		client.MatchingLabels(collector.Spec.Selector.MatchLabels)); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if folder does not exist
	// its assumed that is the first time the collector is being created
	if !r.checkFolderExists(collector) {
		if err := r.createCollectorDir(collector); err != nil {
			log.Error(err, "unable to create collector directory")
			return ctrl.Result{}, err
		}
	}

	now := time.Now()
	// will return the last modified time of the default.pgo file
	// if the file does not exist it returns the zero time
	// so we will force the collection to happen
	lastCollection := r.getLastCollection(collector)
	schedule, err := cron.ParseStandard(collector.Spec.Schedule)
	if err != nil {
		log.Error(err, "unable to parse schedule")
		return ctrl.Result{}, err
	}

	collector.Status.LastCollectionTime = &metav1.Time{Time: now}
	nextSchedule := schedule.Next(lastCollection)

	if nextSchedule.After(now) {
		collector.Status.NextCollectionTime = &metav1.Time{Time: nextSchedule}
		if err := r.Status().Update(ctx, &collector); err != nil {
			log.Error(err, "unable to update PprofCollector status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: now.Sub(nextSchedule),
		}, nil
	}

	collector.Status.NextCollectionTime = &metav1.Time{Time: schedule.Next(now)}
	collector.Status.Phase = collectorv1.Running
	collector.Status.LastCollectedPodsCount = int32(len(pods.Items))
	if err := r.Status().Update(ctx, &collector); err != nil {
		log.Error(err, "unable to update PprofCollector status")
		return ctrl.Result{}, err
	}

	duration, err := time.ParseDuration(collector.Spec.Duration)
	if err != nil {
		log.Error(err, "unable to parse duration")
		return ctrl.Result{}, err
	}

	g := errgroup.Group{}
	podProfiles := []io.Reader{}
	mut := sync.Mutex{}
	for _, pod := range pods.Items {
		g.Go(func() error {
			buf := bytes.Buffer{}
			url := fmt.Sprintf("http://%s:%d%s", pod.Status.PodIP, collector.Spec.Port, collector.Spec.BasePath)
			err = r.PprofCollector.GetProfile(url, duration, &buf)
			log.Info("collecting profile", "url", url)
			if err != nil {
				log.Error(err, "unable to collect profile")
				return err
			}
			mut.Lock()
			podProfiles = append(podProfiles, &buf)
			mut.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		collector.Status.Phase = collectorv1.Failed
		collector.Status.Message = "Failed to collect profiles"
		if err := r.Status().Update(ctx, &collector); err != nil {
			log.Error(err, "unable to update PprofCollector status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if len(podProfiles) == 0 {
		collector.Status.Phase = collectorv1.Failed
		collector.Status.Message = "No profiles collected"

	}
	collector.Status.Phase = collectorv1.Succeeded
	collector.Status.Message = "Successfully collected profiles"
	// merge profiles
	mergedFile, err := os.Create(fmt.Sprintf("%s/%s/%s", baseDir, collector.Name, defaultProfileName))
	if err != nil {
		log.Error(err, "unable to create merged file")
		collector.Status.Phase = collectorv1.Failed
		collector.Status.Message = "Failed to created merged file"
	}
	if err := r.PprofCollector.MergeProfiles(podProfiles, mergedFile); err != nil {
		log.Error(err, "unable to merge profiles")
		collector.Status.Phase = collectorv1.Failed
		collector.Status.Message = "Failed to merge profiles"
	}

	if err := r.Status().Update(ctx, &collector); err != nil {
		log.Error(err, "unable to update PprofCollector status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: now.Sub(nextSchedule),
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PprofCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&collectorv1.PprofCollector{}).
		Complete(r)
}

// getLastCollection returns the last modified time of the default.pgo file
// if the file does not exist it returns the zero time
func (r *PprofCollectorReconciler) getLastCollection(collector collectorv1.PprofCollector) time.Time {
	// check the last modified time of the file
	stat, err := os.Stat(fmt.Sprintf("%s/%s/%s", baseDir, collector.Name, defaultProfileName))
	if err != nil {
		return time.Time{}
	}
	return stat.ModTime()
}

// checkFolderExists checks if the folder for the collector exists
func (r *PprofCollectorReconciler) checkFolderExists(collector collectorv1.PprofCollector) bool {
	_, err := os.Stat(fmt.Sprintf("%s/%s", baseDir, collector.Name))
	return !os.IsNotExist(err)
}

// createCollectorDir creates the directory for the collector
func (r *PprofCollectorReconciler) createCollectorDir(collector collectorv1.PprofCollector) error {
	// Create the base directory for the collector
	err := os.MkdirAll(fmt.Sprintf("/%s/%s", baseDir, collector.Name), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
