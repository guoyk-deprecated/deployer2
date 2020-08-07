# deployer2

`deployer` 的船新版本

## 从一代迁移到二代

* 修改 `deployer.yml`

    ```yaml
    version: 2 # 1.必须添加这一行
    default:   # 2.必须有 default:
      build:
        - mvn -P{{.Vars.env}} -Dmaven.test.skip=true clean package # 使用标准 Go {{}} 模板
      package:
        - FROM common-runtime:java8
        - ADD target/xxxx.jar starter.jar
  
    staging:
      vars:
        env: uat # 此值会被 {{.Vars.env}} 引用到
    ```
  
   * 可用的模板变量 
   
       * `.Env` 环境变量，比如 `{{.Env.WORKSPACE}}`
       * `.Profile` 当前环境名
       * `.Vars` 各个环境的 `vars` 字段，比如 `{{.Vars.env}}`
   
   * 可用的模板函数，参考 https://github.com/guoyk93/tmplfuncs
   
* 修改构建命令

  ```shell script
  deployer2 --workload test-qcloud/some-namespace/deployment/some-workload \
            --workload test-qcloud/another-namespace/deployment/another-workload
  # workload 格式 "集群名/命名空间/类型/工作负载名"
  ```

## 许可证

Guo Y.K., MIT License
