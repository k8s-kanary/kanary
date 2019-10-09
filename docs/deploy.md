# 怎样开发 kanary

## 安装 kruise

### 安装 helm 

[https://github.com/openkruise/kruise/blob/master/docs/tutorial/helm-install.md](https://github.com/openkruise/kruise/blob/master/docs/tutorial/helm-install.md)

```yaml
# wget https://cloudnativeapphub.oss-cn-hangzhou.aliyuncs.com/helm-v3.0.0-alpha.2-linux-386.tar.gz
# tar -zxvf helm-v3.0.0-alpha.2-linux-386.tar.gz
```

## 安装 kruise 

参考 kruise 项目文档：[https://github.com/openkruise/kruise/blob/master/docs/tutorial/kruise-install.md](https://github.com/openkruise/kruise/blob/master/docs/tutorial/kruise-install.md)

# 编译 kanary 

```bash
# git clone https://github.com/k8s-kanary/kanary
# make build
# cd build
# docker build . -t registry.cn-hangzhou.aliyuncs.com/k8s-kanary/kanary
# docker push [image id]
```

针对 mac 电脑的交叉编译

```bash
CGO_ENABLED=0 GO111MODULE=on go build GOOS=linux -mod vendor -i -installsuffix cgo -ldflags '-w' -o build/operator ./cmd/manager/main.go
```

# 安装 kanary 

```bash
# install crd
$ kubectl apply -f deploy/crds/kanary_v1alpha1_kanarystatefulset_crd.yaml

# install rbac
$ kubectl apply -f deploy/service_account.yaml
$ kubectl apply -f deploy/role.yaml
$ kubectl apply -f deploy/role_binding.yaml

# install operator
$ kubectl apply -f deploy/operator.yaml
```

