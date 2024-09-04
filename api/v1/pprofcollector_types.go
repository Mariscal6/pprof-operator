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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PprofCollectorSpec defines the desired state of PprofCollector
type PprofCollectorSpec struct {
	// Interval defines how often the pprof data should be collected.
	// use cron format
	Schedule string `json:"schedule,omitempty"`

	// Duration defines the length of time to collect data during each collection period.
	// this cannot be bigger than the interval
	// use golang duration format (e.g., 1m, 1h)
	// +optional
	Duration string `json:"duration,omitempty"`

	// Selector specifies the target pods to collect pprof data from.
	Selector *metav1.LabelSelector `json:"selector"`

	// Port specifies the port to connect to the target pods.
	Port int32 `json:"port,omitempty"`

	// BasePath specifies the base path to collect pprof data from.
	// If not specified /debug/pprof/ will be used.
	// +optional
	BasePath string `json:"basePath,omitempty"`
}

// PprofCollectorStatus defines the observed state of PprofCollector
type PprofCollectorStatus struct {
	// LastCollectionTime is the timestamp of the last successful profile collection.
	LastCollectionTime *metav1.Time `json:"lastCollectionTime,omitempty"`

	// NextCollectionTime is the timestamp of the next scheduled profile collection.
	NextCollectionTime *metav1.Time `json:"nextCollectionTime,omitempty"`

	// LastCollectionStatus indicates whether the last profile collection was successful or failed.
	LastCollectionStatus string `json:"lastCollectionStatus,omitempty"`

	// LastCollectedPodsCount is the number of profiles successfully collected since the PprofCollector started.
	LastCollectedPodsCount int32 `json:"LastCollectedPodsCount,omitempty"`

	// Phase indicates the current phase of the PprofCollector (e.g., Running, Succeeded, Failed).
	Phase CollerctorPhase `json:"phase,omitempty"`

	// Message is a human-readable message indicating details about the current status.
	Message string `json:"message,omitempty"`
}

// CollerctorPhase defines the phase of the PprofCollector.
// +kubebuilder:validation:Enum=Running;Succeeded;Failed
type CollerctorPhase string

const (
	// Pending means the PprofCollector has been accepted by the system, but the collection has not started yet.
	// Running means the collection is in progress.
	Running CollerctorPhase = "Running"

	// Succeeded means the collection has completed successfully.
	Succeeded CollerctorPhase = "Succeeded"

	// Failed means the collection has failed.
	Failed CollerctorPhase = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PprofCollector is the Schema for the pprofcollectors API
type PprofCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PprofCollectorSpec   `json:"spec,omitempty"`
	Status PprofCollectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PprofCollectorList contains a list of PprofCollector
type PprofCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PprofCollector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PprofCollector{}, &PprofCollectorList{})
}
