


helmAppName=inject1
Space=webhookmini

# test2: build-template
# 	echo "test2"


all:codeandimage uninstall build-template install
	sleep 5; make checkrunok;


install:
	- kubectl create ns ${Space}
	helm install webhook-inject-sidecar/ --namespace  ${Space} --values ./values.yaml --name-template ${helmAppName} 

# 清理历史数据
uninstall:
	- helm uninstall --namespace  ${Space} ${helmAppName} 
	# - kubectl delete ns ${Space} 

deletens:
	- kubectl delete ns ${Space} 


build-template:
	rm -Rf template-out-${helmAppName}
	touch values.yaml
	helm template webhook-inject-sidecar/ --namespace  ${Space} --values ./values.yaml --name-template ${helmAppName} --output-dir template-out-${helmAppName} --debug

# 编译linux二进制，和制作docker镜像
codeandimage:
	cd code-webhook-mini && make

# 检查资源
check:
	kubectl  get ValidatingWebhookConfiguration ${helmAppName}-webhook-inject-sidecar-admission -oyaml
	kubectl  get MutatingWebhookConfiguration ${helmAppName}-webhook-inject-sidecar-admission -oyaml
	kubectl -n  ${Space} get secret ${helmAppName}-webhook-inject-sidecar-admission -oyaml

# 部署负载，检测功能是否正常
checkrunok:
	- kubectl delete ns ns12
	- kubectl create ns ns12
	- kubectl label namespace ns12 webhook-mini=enabled
	- kubectl create -n ns12 deployment dep1 --image=nginx --replicas=1 # 期望失败，因为没有相应的label
	- kubectl delete -f dep2.yaml;kubectl apply -f dep2.yaml # 期望成功 kubectl get po -n ns12

getpkifile:
	@echo "cert"
	kubectl get secret -n  ${Space} ${helmAppName}-webhook-inject-sidecar-admission -oyaml |grep cert |awk '{print $$2}'|base64 -d
	@echo "key"
	kubectl get secret -n  ${Space} ${helmAppName}-webhook-inject-sidecar-admission -oyaml |grep key |awk '{print $$2}'|base64 -d

# helm包打包
helmpack:
	helm package webhook-inject-sidecar  # helm仓库配置
	helm repo index .
	git commit -a -m "helm包打包"
	git push origin master
	git checkout  gh-pages
	git merge master
	git push origin gh-pages
	git checkout master # 切回master


	