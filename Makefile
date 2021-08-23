
# Image URL to use all building/pushing image targets
IMG ?= registry.baidubce.com/paddle-operator/controller
SAMPLESET_IMG ?= registry.baidubce.com/paddle-operator/sampleset
SAMPLEJOB_IMG ?= registry.baidubce.com/paddle-operator/samplejob
MANAGER_IMG ?= registry.baidubce.com/paddle-operator/manager
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:maxDescLen=0,generateEmbeddedObjectMeta=true,trivialVersions=true,preserveUnknownFields=false"

# Set version and get git tag
VERSION=v0.3.0
# GIT_SHA=$(shell git rev-parse --short HEAD || echo "HEAD")
# GIT_VERSION=${VERSION}-${GIT_SHA}
GIT_VERSION=${VERSION}

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager gen-deploy

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	#$(KUSTOMIZE) build config/crd | kubectl apply -f -
	$(KUSTOMIZE) build config/crd | kubectl create -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

gen-deploy: manifests kustomize crd-v1beta1
	cat COPYRIGHT.YAML > deploy/v1/crd.yaml
	$(KUSTOMIZE) build config/crd >> deploy/v1/crd.yaml
	cat COPYRIGHT.YAML > deploy/v1/operator.yaml
	$(KUSTOMIZE) build config/operator >> deploy/v1/operator.yaml
	cat COPYRIGHT.YAML > deploy/extensions/controllers.yaml
	$(KUSTOMIZE) build config/extensions >> deploy/extensions/controllers.yaml
	cat COPYRIGHT.YAML config/crd/v1beta1/batch.paddlepaddle.org_paddlejobs.yaml > deploy/v1beta1/crd.yaml
	cat COPYRIGHT.YAML > deploy/v1beta1/operator.yaml
	$(KUSTOMIZE) build config/operator >> deploy/v1beta1/operator.yaml

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}:${VERSION}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

TEMPLATES_DIR ?= charts/paddle-operator/templates
helm: manifests kustomize
	$(KUSTOMIZE) build config/crd > $(TEMPLATES_DIR)/crd.yaml
	$(KUSTOMIZE) build config/default > $(TEMPLATES_DIR)/controller.yaml
	# The last sed command is not clean, need more advanced scripting to be robust
	sed -i  -e "s/image:.*/image: {{ .Values.image }}/g" \
			-e "s/--namespace=.*/--namespace={{ .Values.jobnamespace }}/g" \
			-e "s/namespace:.*/namespace: {{ .Values.controllernamespace }}/g" \
			-e "s/name: paddle-system/name: {{ .Values.controllernamespace }}/g" \
			$(TEMPLATES_DIR)/controller.yaml

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

crd-v1beta1:
	$(CONTROLLER_GEN) "crd:crdVersions=v1beta1,allowDangerousTypes=false,maxDescLen=0,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/v1beta1

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

##@ Build

# Build all controller manager
build-all: manager build-sampleset

# Build sampleset controller manager
build-sampleset: generate fmt vet
	go build -o bin/sampleset-controller cmd/sampleset/main.go

# Build paddlejob controller image
docker-build: test
	docker build . -t ${IMG}:${GIT_VERSION}

# Build all controller manager docker images
docker-build-all: docker-build docker-build-sampleset docker-build-samplejob docker-build-manager

# Build sampleset controller image
docker-build-sampleset: test
	docker build . --build-arg MANAGER_IMG=${MANAGER_IMG} --build-arg GIT_VERSION=${GIT_VERSION} \
		-f docker/Dockerfile.sampleset -t ${SAMPLESET_IMG}:${GIT_VERSION}

# Build samplejob controller image
docker-build-samplejob: test
	docker build . -f docker/Dockerfile.samplejob -t ${SAMPLEJOB_IMG}:${GIT_VERSION}

# Build cache manager runtime image
docker-build-manager: test
	docker build . -f docker/Dockerfile.manager -t ${MANAGER_IMG}:${GIT_VERSION}

# Push the docker image
docker-push:
	docker push ${IMG}:${GIT_VERSION}

# Push all docker images
docker-push-all: docker-push docker-push-sampleset docker-push-samplejob docker-push-manager

# Push the sampleset docker image
docker-push-sampleset:
	docker push ${SAMPLESET_IMG}:${GIT_VERSION}

docker-push-samplejob:
	docker push ${SAMPLEJOB_IMG}:${GIT_VERSION}

docker-push-manager:
	docker push ${MANAGER_IMG}:${GIT_VERSION}

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.1)

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
