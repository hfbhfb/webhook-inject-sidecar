package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	admissionWebhookAnnotationStatusKey = "webhook-example.github.com/status"
	admissionWebhookLabelMutateKey      = "webhook-example.github.com/app"
)

var (
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}

	addLabels = map[string]string{
		admissionWebhookLabelMutateKey: "true",
	}

	addAnnotations = map[string]string{
		admissionWebhookAnnotationStatusKey: "mutated",
	}
)

// 处理逻辑
func mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	req := ar.Request
	var (
		// objectMeta                      *metav1.ObjectMeta
		// resourceNamespace, resourceName string
		deployment appsv1.Deployment
	)

	log.Infof(fmt.Sprintf("======begin Admission for Namespace=[%v], Kind=[%v], Name=[%v]======", req.Namespace, req.Kind.Kind, req.Name))

	switch req.Kind.Kind {
	// 支持Deployment
	case "Deployment":
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil { // 在这里获取deployment
			log.Errorln(fmt.Sprintf("\nCould not unmarshal raw object: %v", err))
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		_, _, _, deployment = deployment.Name, deployment.Namespace, &deployment.ObjectMeta, deployment
	//其他不支持的类型
	default:
		msg := fmt.Sprintf("\nNot support for this Kind of resource  %v", req.Kind.Kind)
		log.Warnf(msg)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: msg,
			},
		}
	}

	//开始处理
	// patchBytes, err := createPatch(deployment, addAnnotations, addLabels)
	patchBytes, err := createPatch(deployment, addAnnotations, nil)
	if err != nil {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Debugf(fmt.Sprintf("AdmissionResponse: patch=%v\n", string(patchBytes)))
	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func mutateHandler(w http.ResponseWriter, r *http.Request) {

	//读取从ApiServer过来的数据放到body
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		logString := "empty body"
		log.Warnf(logString)
		//返回状态码400
		//如果在Apiserver调用此Webhook返回是400，说明APIServer自己传过来的数据是空
		http.Error(w, logString, http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logString := fmt.Sprintf("Content-Type=%s, expect `application/json`", contentType)
		log.Warnf(logString)
		//如果在Apiserver调用此Webhook返回是415，说明APIServer自己传过来的数据不是json格式，处理不了
		http.Error(w, logString, http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *admissionv1.AdmissionResponse
	ar := admissionv1.AdmissionReview{}
	if _, _, err := decoder.Decode(body, nil, &ar); err != nil {
		//组装错误信息
		logString := fmt.Sprintf("\nCan't decode body,error info is :  %s", err.Error())
		log.Errorln(logString)
		//返回错误信息，形式表现为资源创建会失败，
		admissionResponse = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: logString,
			},
		}
	} else {
		if r.URL.Path == "/mutate" {
			admissionResponse = mutate(&ar)

			ar.Response = admissionResponse
			// 设置 AdmissionReview 的 UID 和 API 版本
			ar.Response.UID = ar.Request.UID
			ar.APIVersion = admissionv1.SchemeGroupVersion.String()

			resp, err := json.Marshal(ar)
			if err != nil {
				logString := fmt.Sprintf("\nCan't encode response: %v", err)
				log.Errorln(logString)
				http.Error(w, logString, http.StatusInternalServerError)
			}
			log.Infoln("Ready to write reponse ...")
			if _, err := w.Write(resp); err != nil {
				logString := fmt.Sprintf("\nCan't write response: %v", err)
				log.Errorln(logString)
				http.Error(w, logString, http.StatusInternalServerError)
			}

			//东八区时间
			datetime := time.Now().In(time.FixedZone("GMT", 8*3600)).Format("2006-01-02 15:04:05")
			logString := fmt.Sprintf("======%s ended Admission already writed to reponse======", datetime)
			//最后打印日志
			log.Infof(logString)
		}
	}
}

// 拼接PatchJson
func createPatch(deployment appsv1.Deployment, addAnnotations map[string]string, addLabels map[string]string) ([]byte, error) {

	var patches []patchOperation
	objectMeta := deployment.ObjectMeta
	// labels := objectMeta.Labels
	annotations := objectMeta.Annotations
	// labelsPatch := updateLabels(labels, addLabels) //此处拼接有些异常，需再定位。（因为 validate webhook会检查label）
	annotationsPatch := updateAnnotation(annotations, addAnnotations)
	containersPatch := updateContainers(addContainer, deployment)

	// patches = append(patches, labelsPatch...)
	patches = append(patches, annotationsPatch...)
	patches = append(patches, containersPatch...)

	return json.Marshal(patches)
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func updateLabels(target map[string]string, added map[string]string) (patch []patchOperation) {
	values := make(map[string]string)
	for key, value := range added {
		if target == nil || target[key] == "" {
			values[key] = value
		}
	}
	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: values,
	})
	return patch
}

var addContainer = []corev1.Container{
	{
		Name:            "side-car",
		Image:           "busybox",
		Command:         []string{"/bin/sleep", "infinity"},
		ImagePullPolicy: "IfNotPresent",
	},
}

func updateContainers(addContainer []corev1.Container, deployment appsv1.Deployment) (patch []patchOperation) {
	currentDeployment := deployment.DeepCopy()
	containers := currentDeployment.Spec.Template.Spec.Containers
	containers = append(containers, addContainer...)
	currentDeployment.Spec.Template.Spec.Containers = containers
	diffPatch, err := jsondiff.Compare(deployment, currentDeployment)
	if err != nil {
		log.Error("")
	}
	for _, v := range diffPatch {
		addPatch := patchOperation{
			Op:    v.Type,
			Value: v.Value,
			Path:  string(v.Path),
		}
		patch = append(patch, addPatch)
	}
	return patch
}
