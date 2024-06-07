package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"

	log "github.com/sirupsen/logrus"

	// "github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	port     = 443
	certFile = "./pki/cert" // 证书文件路径
	keyFile  = "./pki/key"  // 私钥文件路径
)

var (
	// 定义反序列化器
	decoder runtime.Decoder
	// 定义要求的标签
	requiredLabel = "required-label"
)

func init() {
	// 创建反序列化器
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	decoder = codecs.UniversalDeserializer()
	log.SetLevel(log.DebugLevel)
}
func main() {
	// 注册处理函数
	http.HandleFunc("/validate", validateHandler)
	http.HandleFunc("/mutate", mutateHandler)
	// 加载证书和私钥文件
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to load certificate and key: %v", err)
	}
	// 创建 TLS 配置
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	// 创建 HTTPS 服务器
	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", port),
		TLSConfig: tlsConfig,
	}
	// 启动 Web 服务器
	log.Printf("Server listening on port %v", port)
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	// 解析 admissionReview 请求对象
	admissionReview := admissionv1.AdmissionReview{}
	_, _, err = decoder.Decode(data, nil, &admissionReview)
	if err != nil {
		log.Printf("Failed to decode AdmissionReview: %v", err)
		http.Error(w, "Failed to decode AdmissionReview", http.StatusBadRequest)
		return
	}
	pod := corev1.Pod{}
	err = json.Unmarshal(admissionReview.Request.Object.Raw, &pod)
	if err != nil {
		log.Printf("Failed to unmarshal Pod: %v", err)
		http.Error(w, "Failed to unmarshal Pod", http.StatusBadRequest)
		return
	}
	if !validatePodLabels(&pod) {
		// 标签不符合要求，返回错误响应
		admissionReview.Response = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				// 自定义状态码和返回客户端的信息
				Message: "Pod labels do not meet the requirements",
				Code:    403,
			},
			// 通过Allowed字段控制允许请求或禁止请求
			Allowed: false,
		}

	} else {
		// ValidatingAdmissionWebhook和MutatingAdmissionWebhook的区别就在于
		// 处理Mutating Webhook时需要拼接JSONPatch 的数据
		// 当没有此处逻辑时，该示例代码就是个验证性的webhook

		// patchTypeConst := admissionv1.PatchTypeJSONPatch
		// admissionReview.Response = &admissionv1.AdmissionResponse{
		// 	Allowed:   true,
		// 	PatchType: &patchTypeConst,
		// 	// 当传入指定标签时，通过patch操作修改镜像
		// 	Patch: patchOperationFun(),
		// }

		admissionReview.Response = &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}
	// 设置 AdmissionReview 的 UID 和 API 版本
	admissionReview.Response.UID = admissionReview.Request.UID
	admissionReview.APIVersion = admissionv1.SchemeGroupVersion.String()
	// 序列化 AdmissionReview 对象
	responseData, err := json.Marshal(admissionReview)
	if err != nil {
		log.Printf("Failed to marshal AdmissionReview: %v", err)
		http.Error(w, "Failed to marshal AdmissionReview", http.StatusInternalServerError)
		return
	}
	// 返回响应数据
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(responseData)
	if err != nil {
		log.Printf("Failed to write response: %v", err)
	}

}

func validatePodLabels(pod *corev1.Pod) bool {
	// return true
	for key, _ := range pod.Labels {
		if key == requiredLabel {
			return true
		}
	}

	return false
}

func patchOperationFun() []byte {
	str := `[{
		"op":    "replace",
		"path":  "/spec/containers/0/image",
		"value": "nginx:1.16"
	}]`

	return []byte(str)
}
