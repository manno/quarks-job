## quarks-job version

Print the version number

### Synopsis

Print the version number

```
quarks-job version [flags]
```

### Options

```
  -h, --help   help for version
```

### Options inherited from parent commands

```
      --apply-crd                         (APPLY_CRD) If true, apply CRDs on start (default true)
      --ctx-timeout int                   (CTX_TIMEOUT) context timeout for each k8s API request in seconds (default 30)
  -o, --docker-image-org string           (DOCKER_IMAGE_ORG) Dockerhub organization that provides the operator docker image (default "cfcontainerization")
      --docker-image-pull-policy string   (DOCKER_IMAGE_PULL_POLICY) Image pull policy (default "IfNotPresent")
  -r, --docker-image-repository string    (DOCKER_IMAGE_REPOSITORY) Dockerhub repository that provides the operator docker image (default "cf-operator")
  -t, --docker-image-tag string           (DOCKER_IMAGE_TAG) Tag of the operator docker image
  -c, --kubeconfig string                 (KUBECONFIG) Path to a kubeconfig, not required in-cluster
  -l, --log-level string                  (LOG_LEVEL) Only print log messages from this level onward (default "debug")
      --max-workers int                   (MAX_WORKERS) Maximum number of workers concurrently running the controller (default 1)
  -n, --operator-namespace string         (OPERATOR_NAMESPACE) The operator namespace (default "default")
      --watch-namespace string            (WATCH_NAMESPACE) Namespace to watch for BOSH deployments
```

### SEE ALSO

* [quarks-job](quarks-job.md)	 - quarks-job starts the operator

###### Auto generated by spf13/cobra on 16-Oct-2019