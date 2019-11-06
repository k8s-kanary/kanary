package comparison

import (
	"bytes"
	"crypto/md5" // #nosec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	apps "k8s.io/api/apps/v1beta1"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
)

// 针对statefulset的再写一套函数。
func IsUpToDateStatefulset(kd *kanaryv1alpha1.KanaryStatefulset, sts *apps.StatefulSet) bool {
	hash, err := GenerateMD5StatefulsetSpec(&kd.Spec.Template.Spec)
	if err != nil {
		return false
	}
	return CompareStatefulsetMD5Hash(hash, sts)
}

// CompareStatefulsetMD5Hash used to compare a md5 hash with the one setted in Deployment annotation
func CompareStatefulsetMD5Hash(hash string, sts *apps.StatefulSet) bool {
	if val, ok := sts.Annotations[string(kanaryv1alpha1.MD5KanaryStatefulsetAnnotationKey)]; ok && val == hash {
		return true
	}
	return false
}

// GenerateMD5DeploymentSpec used to generate the DeploymentSpec MD5 hash
func GenerateMD5StatefulsetSpec(spec *apps.StatefulSetSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	/* #nosec */
	hash := md5.New()
	_, err = io.Copy(hash, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SetMD5StatefulsetSpecAnnotation used to set the md5 annotation key/value from the KanaryDeployement.Spec.Template.Spec
func SetMD5StatefulsetSpecAnnotation(kd *kanaryv1alpha1.KanaryStatefulset, sts *apps.StatefulSet) (string, error) {
	md5Spec, err := GenerateMD5StatefulsetSpec(&kd.Spec.Template.Spec)
	if err != nil {
		return "", fmt.Errorf("unable to generates the JobSpec MD5, %v", err)
	}
	if sts.Annotations == nil {
		sts.SetAnnotations(map[string]string{})
	}

	sts.Annotations[string(kanaryv1alpha1.MD5KanaryStatefulsetAnnotationKey)] = md5Spec
	return md5Spec, nil
}