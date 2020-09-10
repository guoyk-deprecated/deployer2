package main

import "path"

type Patch struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	} `json:"metadata,omitempty"`
	Spec struct {
		Template struct {
			Metadata struct {
				Annotations struct {
					Timestamp string `json:"net.guoyk.deployer/timestamp,omitempty"`
				} `json:"annotations"`
			} `json:"metadata"`
			Spec struct {
				Containers       []PatchContainer       `json:"containers,omitempty"`
				InitContainers   []PatchInitContainer   `json:"initContainers,omitempty"`
				ImagePullSecrets []PatchImagePullSecret `json:"imagePullSecrets,omitempty"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
}

type PatchContainer struct {
	Image           string `json:"image"`
	Name            string `json:"name"`
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
	Resources       struct {
		Limits struct {
			CPU    string `json:"cpu,omitempty"`
			Memory string `json:"memory,omitempty"`
		} `json:"limits,omitempty"`
		Requests struct {
			CPU    string `json:"cpu,omitempty"`
			Memory string `json:"memory,omitempty"`
		} `json:"requests,omitempty"`
	} `json:"resources,omitempty"`
}

type PatchInitContainer struct {
	Image           string `json:"image"`
	Name            string `json:"name"`
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}

type PatchImagePullSecret struct {
	Name string `json:"name"`
}

type ImageNames []string

func (ims ImageNames) Primary() string {
	return ims[0]
}

func (ims ImageNames) Derive(registry string) ImageNames {
	out := make(ImageNames, len(ims), len(ims))
	for i, im := range ims {
		out[i] = path.Join(registry, im)
	}
	return out
}
