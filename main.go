package main

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/guoyk93/deployer2/pkg/cmds"
	"github.com/guoyk93/deployer2/pkg/image_tracker"
	"github.com/guoyk93/tempfile"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

		imageNames   ImageNames
		imageTracker = image_tracker.New()
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

	_ = cmds.DockerVersion()

	var m Manifest
	log.Printf("清单文件: %s", optManifest)
	if m, err = LoadManifestFile(optManifest); err != nil {
		return
	}

	log.Printf("使用环境: %s", optProfile)
	f := m.Profile(optProfile)
	var fileBuild, filePackage string
	if fileBuild, filePackage, err = f.GenerateFiles(); err != nil {
		return
	}
	log.Printf("写入构建文件: %s", fileBuild)
	log.Printf("写入打包文件: %s", filePackage)

	log.Println("------------ 构建 ------------")
	if err = cmds.Execute(fileBuild); err != nil {
		return
	}
	log.Println("构建完成")

	log.Println("------------ 打包 ------------")
	if err = cmds.DockerBuild(filePackage, imageNames.Primary()); err != nil {
		return
	}

	log.Printf("打包完成: %s", imageNames.Primary())

	imageTracker.Add(imageNames.Primary())
	defer imageTracker.DeleteAll()

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

		var dcDir, kcFile string
		if dcDir, kcFile, err = s.GenerateFiles(); err != nil {
			return
		}

		_ = cmds.KubectlVersion(kcFile)

		remoteImageNames := imageNames.Derive(s.Registry)

		for _, remoteImageName := range remoteImageNames {
			log.Printf("推送镜像: %s", remoteImageName)
			if err = cmds.DockerTag(imageNames.Primary(), remoteImageName); err != nil {
				return
			}
			imageTracker.Add(remoteImageName)
			if err = cmds.DockerPush(remoteImageName, dcDir); err != nil {
				return
			}
		}

		if optSkipDeploy {
			continue
		}

		// 构建 Patch
		var p UniversalPatch
		p.Metadata.Annotations = s.Annotations
		if p.Metadata.Annotations == nil {
			p.Metadata.Annotations = map[string]string{}
		}
		if p.Spec.Template.Annotations == nil {
			p.Spec.Template.Annotations = map[string]string{}
		}
		p.Spec.Template.Annotations["net.guoyk.deployer/timestamp"] = time.Now().Format(time.RFC3339)
		for _, name := range s.ImagePullSecrets {
			secret := corev1.LocalObjectReference{Name: strings.TrimSpace(name)}
			p.Spec.Template.Spec.ImagePullSecrets = append(p.Spec.Template.Spec.ImagePullSecrets, secret)
		}
		if workload.IsInit {
			container := corev1.Container{
				Image:           remoteImageNames.Primary(),
				Name:            workload.Container,
				ImagePullPolicy: "Always",
			}
			p.Spec.Template.Spec.InitContainers = append(p.Spec.Template.Spec.InitContainers, container)
		} else {
			container := corev1.Container{
				Image:           remoteImageNames.Primary(),
				Name:            workload.Container,
				ImagePullPolicy: "Always",
			}
			if container.Resources.Requests == nil {
				container.Resources.Requests = map[corev1.ResourceName]resource.Quantity{}
			}
			if container.Resources.Limits == nil {
				container.Resources.Limits = map[corev1.ResourceName]resource.Quantity{}
			}
			cpu, mem := s.CPU, s.MEM
			if f.CPU != nil {
				cpu = f.CPU
			}
			if f.MEM != nil {
				mem = f.MEM
			}
			if !optCPU.IsZero() {
				cpu = &optCPU
			}
			if !optMEM.IsZero() {
				mem = &optMEM
			}
			if cpu != nil {
				container.Resources.Requests[corev1.ResourceCPU],
					container.Resources.Limits[corev1.ResourceCPU] = cpu.AsCPU()
			}
			if mem != nil {
				container.Resources.Requests[corev1.ResourceMemory],
					container.Resources.Limits[corev1.ResourceMemory] = mem.AsMEM()
			}
			p.Spec.Template.Spec.Containers = append(p.Spec.Template.Spec.Containers, container)
		}

		var buf []byte
		if buf, err = json.MarshalIndent(p, "", "  "); err != nil {
			return
		}

		if err = cmds.KubectlPatch(kcFile, workload.Namespace, workload.Name, workload.Type, string(buf)); err != nil {
			return
		}
	}
}
