# qliksense-operator

WIP

This operator creates patchs for qliksene configs. It expects certain directory structure in the qliksense configs location.

```console
manifests-root
|--.operator
|   |--kustomization.yaml
|   |--configs
|   |  |--kustomization.yaml
|   |--secrets
|   |  |--kustomization.yaml
|   |--patches
|   |  |--kustomization.yaml
|--manifests
|  |--base
|  |  |........
|  |  |--kustomization.yaml
```

It works based on CR config yaml in environment variable `YAML_CONF`. The CR config looks like this

```yaml
configProfile: manifests/base
manifestsRoot: "/cnab/app"
storageClassName: efs
configs:
- dataKey: acceptEULA
  values:
    qliksense: "yes"
secrets:
- secretKey: mongoDbUri
  values:
    qliksense: mongo://mongo:3307
```