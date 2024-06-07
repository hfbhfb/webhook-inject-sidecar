
IMG ?= swr.cn-north-4.myhuaweicloud.com/hfbbg4/webhookmini:v1




test: build-docker deploy-k8s
	echo "test"


build-docker: cross-build
	DOCKER_BUILDKIT=0 docker build  -t ${IMG} .

cross-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/webhookmini 

deploy-k8s:
	-kubectl delete deployment webhookmini
	kubectl create deployment webhookmini --image=${IMG} --replicas=1 

test: cross-build
	echo "all"
