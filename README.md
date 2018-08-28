# A controller service for CSI-compliant local disk

This is [Cloud Foundry](https://github.com/cloudfoundry)'s implementation of the [Container Storage Interface](https://github.com/container-storage-interface/spec/blob/master/spec.md)'s Controller Plugin for local volumes. The [CSI Local Volume Release](https://github.com/cloudfoundry/csi-local-volume-release) for Cloud Foundry submodules this repository. Functionally, this repository enables the CSI Local Volume Release access to controller capabilities and serves to make the release compliant with controller RPCs.

This repository is to be operated solely for testing purposes. It should be used as an example for all other Cloud Foundry controller plugins adhering to the Container Storage Interface.  

# Developer Notes

THIS REPOSITORY IS A WORK IN PROGRESS.

| RPC | Expected Response |
|---|---|
| CreateVolume | Success response with name of the volume created |
| DeleteVolume | Success response |
| ControllerPublishVolume | Empty Response |
| ControllerUnpublishVolume | Empty Response |
| ValidateVolumeCapabilities | True if no capabilities are specified, False if either FsType or mount flags is specified |
| ListVolumes | Empty Response |
| GetCapacity | Empty Response |
| ControllerGetCapabilities | Returns response with all controller capabilities |

Note: Even though CreateVolume and DeleteVolume return a response that a volume is created or deleted, the actual functionality under the hood is a no op. Since we're using a local volume, we designate the [node plugin](https://github.com/cloudfoundry/local-node-plugin) to handle the actual creation and deletion of the plugin.

## Running Tests

1. Install [go](https://golang.org/doc/install).
1. ```export PATH=$GOPATH/bin:$PATH```
1. ```go get code.cloudfoundry.org/local-controller-plugin >/dev/null 2>&1 || true```
1. ```pushd $GOPATH/code.cloudfoundry.org/local-controller-plugin```
1. ```scripts/go_get_all_dep.sh```
1. ```ginkgo -r```
