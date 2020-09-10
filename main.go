package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/guoyk93/tempfile"
	"log"
	"os"
	"strings"
	"time"
)

func exit(err *error) {
	if *err != nil {
		log.Println("错误退出:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("正常退出")
	}
}

func main() {
	var err error
	defer exit(&err)
	defer tempfile.DeleteAll()

	log.SetOutput(os.Stdout)
	log.SetPrefix("[deployer2] ")

	var (
		optManifest   string
		optImage      string
		optProfile    string
		optWorkloads  WorkloadOptions
		optCPU        LimitOption
		optMEM        LimitOption
		optSkipDeploy bool

		imageNames     ImageNames
		usedImageNames ImageNames
	)

	flag.StringVar(&optManifest, "manifest", "deployer.yml", "指定描述文件")
	flag.StringVar(&optImage, "image", "", "镜像名")
	flag.StringVar(&optProfile, "profile", "", "指定环境名")
	flag.BoolVar(&optSkipDeploy, "skip-deploy", false, "跳过部署流程")
	flag.Var(&optWorkloads, "workload", "指定目标工作负载，格式为 \"CLUSTER/NAMESPACE/TYPE/NAME[/CONTAINER]\"")
	flag.Var(&optCPU, "cpu", "指定 CPU 配额，格式为 \"MIN:MAX\"，单位为 m (千分之一核心)")
	flag.Var(&optMEM, "mem", "指定 MEM 配额，格式为 \"MIN:MAX\"，单位为 Mi (兆字节)")
	flag.Parse()

	// 从 JOB_NAME 获取 image 和 profile 信息
	if optImage == "" || optProfile == "" {
		if jobNameSplits := strings.Split(os.Getenv("JOB_NAME"), "."); len(jobNameSplits) == 2 {
			if optImage == "" {
				optImage = jobNameSplits[0]
			}
			if optProfile == "" {
				optProfile = jobNameSplits[1]
			}
		} else {
			err = errors.New("缺少 --image 或者 --profile 参数，且无法从 $JOB_NAME 获得有用信息")
			return
		}
	}

	// 计算标签，第一个标签为主标签
	if buildNumber := os.Getenv("BUILD_NUMBER"); buildNumber != "" {
		imageNames = append(imageNames, optImage+":"+optProfile+"-build-"+buildNumber)
	}
	imageNames = append(imageNames, optImage+":"+optProfile)

	log.Println("------------ deployer2 ------------")

	var m Manifest
	log.Printf("清单文件: %s", optManifest)
	if m, err = LoadManifestFile(optManifest); err != nil {
		return
	}

	log.Printf("使用环境: %s", optProfile)
	var fileBuild, filePackage string
	if fileBuild, filePackage, err = m.Profile(optProfile).GenerateFiles(); err != nil {
		return
	}
	log.Printf("写入构建文件: %s", fileBuild)
	log.Printf("写入打包文件: %s", filePackage)

	log.Println("------------ 构建 ------------")
	if err = Execute(fileBuild); err != nil {
		return
	}
	log.Println("构建完成")

	log.Println("------------ 打包 ------------")
	if err = ExecuteDockerBuild(filePackage, imageNames.Primary()); err != nil {
		return
	}

	log.Printf("打包完成: %s", imageNames.Primary())
	usedImageNames = append(usedImageNames, imageNames.Primary())

	defer func() {
		log.Printf("清理镜像")
		for _, imageName := range usedImageNames {
			_ = ExecuteDockerRemoveImage(imageName)
		}
	}()

	// 执行推送/部署流程
	for _, workload := range optWorkloads {
		log.Printf("------------ 部署 [%s] ------------", workload.String())

		var s Preset
		if s, err = LoadPreset(workload.Cluster); err != nil {
			if os.IsNotExist(err) {
				log.Printf("无法找到集群预置文件 %s, 请确认 --workload 参数是否正确", workload.Cluster)
			}
			return
		}

		fullImageNames := imageNames.Derive(s.Registry)

		var dcDir, dcFile string
		if dcDir, dcFile, err = tempfile.WriteDirFile(
			s.GenerateDockerconfig(),
			"deployer-dockerconfig",
			"config.json",
			false,
		); err != nil {
			return
		}
		log.Printf("生成 Docker 配置文件: %s", dcFile)

		for _, fullImageName := range fullImageNames {
			log.Printf("推送镜像: %s", fullImageName)

			if err = ExecuteDockerTag(imageNames.Primary(), fullImageName); err != nil {
				return
			}

			usedImageNames = append(usedImageNames, fullImageName)

			if err = ExecuteDockerPush(fullImageName, dcDir); err != nil {
				return
			}
		}

		if optSkipDeploy {
			continue
		}

		var fileKubeconfig string
		if fileKubeconfig, err = tempfile.WriteFile(s.GenerateKubeconfig(), "deployer-kubeconfig", ".yml", false); err != nil {
			return
		}
		log.Printf("生成 Kubeconfig 文件: %s", fileKubeconfig)

		// 构建 Patch
		var p Patch
		p.Metadata.Annotations = s.ExtraAnnotations
		p.Spec.Template.Metadata.Annotations.Timestamp = time.Now().Format(time.RFC3339)
		for _, name := range s.ImagePullSecrets {
			secret := PatchImagePullSecret{Name: strings.TrimSpace(name)}
			p.Spec.Template.Spec.ImagePullSecrets = append(p.Spec.Template.Spec.ImagePullSecrets, secret)
		}
		if workload.IsInit {
			container := PatchInitContainer{
				Image:           fullImageNames.Primary(),
				Name:            workload.Container,
				ImagePullPolicy: "Always",
			}
			p.Spec.Template.Spec.InitContainers = append(p.Spec.Template.Spec.InitContainers, container)
		} else {
			container := PatchContainer{
				Image:           fullImageNames.Primary(),
				Name:            workload.Container,
				ImagePullPolicy: "Always",
			}
			container.Resources.Requests.CPU = s.RequestsCPU
			container.Resources.Requests.Memory = s.RequestsMEM
			container.Resources.Limits.CPU = s.LimitsCPU
			container.Resources.Limits.Memory = s.LimitsMEM
			if !optCPU.IsZero() {
				container.Resources.Requests.CPU = fmt.Sprintf("%dm", optCPU.Min)
				container.Resources.Limits.CPU = fmt.Sprintf("%dm", optCPU.Max)
			}
			if !optMEM.IsZero() {
				container.Resources.Requests.Memory = fmt.Sprintf("%dMi", optMEM.Min)
				container.Resources.Limits.Memory = fmt.Sprintf("%dMi", optMEM.Max)
			}
			p.Spec.Template.Spec.Containers = append(p.Spec.Template.Spec.Containers, container)
		}

		var buf []byte
		if buf, err = json.Marshal(p); err != nil {
			return
		}

		if err = ExecuteKubectlPatch(fileKubeconfig, workload.Namespace, workload.Name, workload.Type, string(buf)); err != nil {
			return
		}
	}
}
