package main

import (
	"encoding/json"
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Preset struct {
	Registry         string                 `yaml:"registry"`
	ImagePullSecrets []string               `yaml:"imagePullSecrets"`
	LimitsCPU        string                 `yaml:"limitsCPU"`
	LimitsMEM        string                 `yaml:"limitsMEM"`
	RequestsCPU      string                 `yaml:"requestsCPU"`
	RequestsMEM      string                 `yaml:"requestsMEM"`
	ExtraAnnotations map[string]string      `yaml:"extra_annotations"`
	Kubeconfig       map[string]interface{} `yaml:"kubeconfig"`
	Dockerconfig     *struct {
		Auths map[string]map[string]string `json:"auths" yaml:"auths"`
	} `yaml:"dockerconfig"`
}

func LoadPreset(cluster string) (p Preset, err error) {
	var home string
	if home = os.Getenv("HOME"); len(home) == 0 {
		err = errors.New("缺少环境变量 $HOME")
		return
	}
	filename := filepath.Join(home, ".deployer", "preset-"+cluster+".yml")
	log.Printf("加载集群配置: %s", filename)
	var buf []byte
	if buf, err = ioutil.ReadFile(filename); err != nil {
		return
	}
	err = yaml.Unmarshal(buf, &p)
	return
}

func (p Preset) GenerateKubeconfig() []byte {
	if p.Kubeconfig == nil {
		return []byte{}
	}
	buf, err := yaml.Marshal(p.Kubeconfig)
	if err != nil {
		panic(err)
	}
	return buf
}

func (p Preset) GenerateDockerconfig() []byte {
	if p.Dockerconfig == nil {
		return []byte("{}")
	}
	buf, err := json.Marshal(p.Dockerconfig)
	if err != nil {
		panic(err)
	}
	return buf
}
