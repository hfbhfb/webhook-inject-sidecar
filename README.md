


## 开发部署操作

```sh
# 进入helm工作目录
cd helm-opt


# 安装
make uninstall
make install

# 查检
make checkrunok


# 查看template编译情况
make build-template

# 获取需要替换pki目录的文件
make getpkifile


# 手动将将 failurePolicy: Ignore 删除
kubectl edit MutatingWebhookConfiguration inject1-webhook-inject-sidecar-admission
kubectl edit ValidatingWebhookConfiguration inject1-webhook-inject-sidecar-admission





```



## 参考的文档
https://blog.csdn.net/weixin_47575974/article/details/132109151





