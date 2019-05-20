# Building KinD node image

Clone https://github.com/kubernetes-sigs/kind.

Update `images/base/entrypoint` in kind repo with
```
mount --make-shared /sys
```
 in `fix_mount()` function, refer: https://github.com/darkowlzz/kind/blob/kind-17-05-19/images/base/entrypoint#L45.

(Optional) Update `pkg/build/base/base.go` with:
```
cmd := exec.Command("docker", "build", "--no-cache", "-t", c.image, dir)
```
in `buildImage()`, to avoid using cache when rebuilding the container images.

Build and publish kind node image:
```
# Download the vendor deps
$ go mod vendor

# Build KinD
$ go build -v sigs.k8s.io/kind

# Build base image with /sys as shared mount
$ ./kind build base-image

# Build node image using the new base-image
$ ./kind build node-image --base-image kindest/base:latest

# Use the new kindest/node image to create a new k8s cluster
$ ./kind create cluster --image kindest/node:latest

# Tag and push the node image:
$ docker tag kindest/node:latest storageos/kind-node:v1.14.2
$ docker push storageos/kind-node:v1.14.2
```
