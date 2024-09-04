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

package v1

import (
	"fmt"
	"net/url"
	"time"

	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var pprofcollectorlog = logf.Log.WithName("pprofcollector-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *PprofCollector) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-collector-profiling-operators-v1-pprofcollector,mutating=true,failurePolicy=fail,sideEffects=None,groups=collector.profiling.operators,resources=pprofcollectors,verbs=create;update,versions=v1,name=mpprofcollector.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &PprofCollector{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PprofCollector) Default() {
	pprofcollectorlog.Info("default", "name", r.Name)
	if r.Spec.Duration == "" {
		r.Spec.Duration = "10s"
	}
	if r.Spec.BasePath == "" {
		r.Spec.BasePath = "/debug/pprof/profile"
	}
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-collector-profiling-operators-v1-pprofcollector,mutating=false,failurePolicy=fail,sideEffects=None,groups=collector.profiling.operators,resources=pprofcollectors,verbs=create;update,versions=v1,name=vpprofcollector.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &PprofCollector{}

func (r *PprofCollector) Validate() (admission.Warnings, error) {
	duration, err := time.ParseDuration(r.Spec.Duration)
	if err != nil {
		return nil, err
	}
	cronPattern, err := cron.ParseStandard(r.Spec.Schedule)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	next := cronPattern.Next(now)

	if cronPattern.Next(next).Sub(next) < duration {
		err = fmt.Errorf("schedule cannot be less than duration")
		logf.Log.Error(err, "schedule duration", "duration", cronPattern.Next(next).Sub(next), "next", next.String(), "afer-next", cronPattern.Next(next).String())
		return nil, err
	}

	_, err = url.Parse(r.Spec.BasePath)
	if err != nil {
		return nil, err
	}

	if r.Spec.Selector == nil || len(r.Spec.Selector.MatchLabels) == 0 {
		return nil, fmt.Errorf("selector is required")
	}

	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PprofCollector) ValidateCreate() (admission.Warnings, error) {
	pprofcollectorlog.Info("validate create", "name", r.Name)
	return r.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PprofCollector) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	pprofcollectorlog.Info("validate update", "name", r.Name)
	return r.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PprofCollector) ValidateDelete() (admission.Warnings, error) {
	pprofcollectorlog.Info("validate delete", "name", r.Name)
	return nil, nil
}
