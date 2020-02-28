# Qliksense Operator

qliksense-operator is used to manage QSEoK resources inside kubernetes cluster. It can be a light-weight git-ops operator as well. This operator is installed by the [sense-installer](https://github.com/qlik-oss/sense-installer)

## Operator Deployment

Any Kubernetes operator has two parts 1. CRD 2. Controller. For qliksense operator, custom resource definition [CRD](deploy/crds/qlik.com_qliksenses_crd.yaml) need to be deployed first. The [sense-installer](https://github.com/qlik-oss/sense-installer) has command to do that (`qliksense opeartor crd install`) but it needs cluster level permission to do that. Then controller part need to be installed. The [sense-installer](https://github.com/qlik-oss/sense-installer) does it automatically and it does not require cluster level permission.

## Operation Mode

The qliksense operator works differently based on if the CR has git repo in it or not. The non-git CR looks like this

```yaml
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: qlik-default
spec:
  profile: docker-desktop
  secrets:
    qliksense:
    - name: mongoDbUri
      value: mongodb://qlik-default-mongodb:27017/qliksense?ssl=false
  configs:
    qliksense:
    - name: acceptEULA
      value: "yes"
  manifestsRoot: /Users/mqb/learning/qliksense-k8s
  rotateKeys: "yes"
  releaseName: qlik-default
```

After installing QSEoK by [sense-installer](https://github.com/qlik-oss/sense-installer) it will create the above CR. Then operator take the owner ship of all the resoruces for QSEoK. So that operator can delete/manage QSEoK resources. This provides the flexibility for customer to switch to git-ops mode by providing the following CR without any service outage

```yaml
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: qlik-default
  labels:
    version: v0.0.2
spec:
  profile: docker-desktop
  git:
    repository: https://github.com/my-org/qliksense-k8s
    accessToken: balallafafafaf
  secrets:
    qliksense:
    - name: mongoDbUri
      value: mongodb://qlik-default-mongodb:27017/qliksense?ssl=false
  configs:
    qliksense:
    - name: acceptEULA
      value: "yes"
  manifestsRoot: /Users/mqb/learning/qliksense-k8s
  rotateKeys: "yes"
  releaseName: qlik-default
```

## Light-Weight git-ops

Having git repo in the CR, the operator can install QSEoK and initiate a cronjob to watch master branch of the repo. Any changes make into the master branch the cron job will apply those changes into the cluster.
