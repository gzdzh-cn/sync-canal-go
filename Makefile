ROOT_DIR    = $(shell pwd)
NAMESPACE   = "default"
DEPLOY_NAME = "sync-canal-go"
DOCKER_NAME = "sync-canal-go"

include ./hack/hack-cli.mk
include ./hack/hack.mk
