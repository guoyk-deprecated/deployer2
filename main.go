package main

import (
	"flag"
	"log"
	"os"
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("exited")
	}
}

func main() {
	var err error
	defer exit(&err)

	log.SetOutput(os.Stdout)
	log.SetPrefix("[deployer] ")

	var (
		optManifest string
		optEnv      string
		optWorkload WorkloadOptions
		optCPU      LimitOption
		optMEM      LimitOption
	)

	flag.StringVar(&optManifest, "manifest", "deployer.yml", "指定描述文件")
	flag.StringVar(&optEnv, "env", "", "指定环境名")
	flag.Var(&optWorkload, "workload", "指定目标工作负载，格式为 \"CLUSTER/NAMESPACE/TYPE/NAME\"")
	flag.Var(&optCPU, "cpu", "指定 CPU 配额，格式为 \"MIN:MAX\"，单位为 m (千分之一核心)")
	flag.Var(&optMEM, "mem", "指定 MEM 配额，格式为 \"MIN:MAX\"，单位为 Mi (兆字节)")
	flag.Parse()
}
