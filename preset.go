package main

import (
	"encoding/json"
	"errors"
	"github.com/guoyk93/tempfile"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Preset struct {
	Registry         string                 `yaml:"registry"`
	Annotations      map[string]string      `yaml:"annotations"`
	ImagePullSecrets []string               `yaml:"imagePullSecrets"`
	CPU              *LimitOption           `yaml:"cpu"`
	MEM              *LimitOption           `yaml:"mem"`
	Kubeconfig       map[string]interface{} `yaml:"kubeconfig"`
	Dockerconfig     map[string]interface{} `yaml:"dockerconfig"`
}

func LoadPreset(cluster string) (p Preset, err error) {
	var home string
	if home = os.Getenv("HOME"); len(home) == 0 {
		err = errors.New("缺少环境变量 $HOME")
		return
	}
	filename := filepath.Join(home, ".deployer2", "preset-"+cluster+".yml")
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

func (p Preset) GenerateFiles() (kcFile string, dcDir string, err error) {
	var dcFile string
	if dcDir, dcFile, err = tempfile.WriteDirFile(
		p.GenerateDockerconfig(),
		"deployer-dockerconfig",
		"config.json",
		false,
	); err != nil {
		return
	}
	log.Printf("生成 Docker 配置文件: %s", dcFile)
	if kcFile, err = tempfile.WriteFile(
		p.GenerateKubeconfig(),
		"deployer-kubeconfig",
		".yml",
		false,
	); err != nil {
		return
	}
	log.Printf("生成 Kubeconfig 文件: %s", kcFile)
	return
}
