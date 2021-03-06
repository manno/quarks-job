# QUARKS JOB

## Introduction

This helm chart deploys the quarks-job operator.

## Installing the latest stable chart

To install the latest stable helm chart, with `quarks-job` as the release name into the namespace `quarks`:

```bash
$ helm install --namespace quarks --name quarks-job https://s3.amazonaws.com/cf-operators/helm-charts/quarks-job-v0.0.1%2B47.g24492ea.tgz
```

## Installing the chart from develop branch

To install the helm chart directly from the [quarks-job repository](https://github.com/cloudfoundry-incubator/quarks-job) (any branch), the following parameters in the `values.yaml` need to be set in advance:


| Parameter                                         | Description                                                          | Default                                        |
| ------------------------------------------------- | -------------------------------------------------------------------- | ---------------------------------------------- |
| `image.repository`                                | docker hub repository for the quarks-job image                      | `quarks-job`                                  |
| `image.org`                                       | docker hub organization for the quarks-job image                    | `cfcontainerization`                           |
| `image.tag`                                       | docker image tag                                                     | `foobar`                                       |


### For a local development with minikube, you can generate the image first and then use the `$ARTIFACT_VERSION` environment variable into the `image.tag`:
```bash
$ eval `minikube docker-env`
$ . bin/include/versioning
$ echo "Tag for docker image is $ARTIFACT_VERSION"
$ bin/build-image
```

Either set the `image.tag` in the `values.yaml`, or pass it to `helm install`:

```bash
$ helm install deploy/helm/quarks-job/ --namespace quarks --name quarks-job --set image.tag=$ARTIFACT_VERSION
```


## Uninstalling the chart

To delete the helm chart:

```bash
To delete the helm chart:

```bash
$ helm delete quarks-job --purge
```

## Configuration

| Parameter                                         | Description                                                                       | Default                                        |
| ------------------------------------------------- | -------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `image.repository`                                | Docker hub repository for the quarks-job image                                         | `quarks-job`                                   |
| `image.org`                                       | Docker hub organization for the quarks-job image                                       | `cfcontainerization`                           |
| `image.tag`                                       | Docker image tag                                                                       | `foobar`                                       |
| `global.contextTimeout`                           | Will set the context timeout in seconds, for future K8S API requests                   | `30`                                           |
| `global.image.pullPolicy`                         | Kubernetes image pullPolicy                                                            | `IfNotPresent`                                 |
| `global.operator.watchNamespace`                  | Namespace the operator will watch for BOSH deployments                                 | the release namespace                          |
| `global.rbac.create`                              | Install required RBAC service account, roles and rolebindings                          | `true`                                         |
| `serviceAccount.create`                           | If true, create a service account                                                      |                                                |
| `serviceAccount.name`                             | If not set and `create` is `true`, a name is generated using the fullname of the chart |                                                |

## RBAC

By default, the helm chart will install RBAC ClusterRole and ClusterRoleBinding based on the chart release name, it will also grant the ClusterRole to an specific service account, which have the same name of the chart release.

The RBAC resources are enable by default. To disable:

```bash
$ helm install --namespace quarks --name quarks-job https://s3.amazonaws.com/cf-operators/helm-charts/quarks-job-v0.2.2%2B47.g24492ea.tgz --set global.rbacEnable=false
```

## Custom Resources

The `quarks-job` watches for the `QuarksJob` custom resource.

The `quarks-job` requires this CRD to be installed in the cluster, in order to work as expected. By default, the `quarks-job` applies the CRD in your cluster automatically.

To verify if the CRD is installed:

```bash
$ kubectl get crds
NAME                                            CREATED AT
quarksjobs.quarks.cloudfoundry.org           2019-06-25T07:08:37Z
```
