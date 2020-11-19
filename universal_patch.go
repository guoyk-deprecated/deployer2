package main

import corev1 "k8s.io/api/core/v1"

type UniversalPatch struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	} `json:"metadata,omitempty"`
	Spec corev1.PodTemplate `json:"spec,omitempty"`
}
