package internal

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	tmaxv1 "github.com/tmax-cloud/approval-watcher/pkg/apis/tmax/v1"
)

const (
	TektonLabelPrefix = "tekton.dev/"
)

func GetApproval(c client.Client, name types.NamespacedName) (*tmaxv1.Approval, error) {
	approval := &tmaxv1.Approval{}
	if err := c.Get(context.TODO(), name, approval); err != nil {
		return nil, err
	}

	return approval, nil
}

func CreateApproval(c client.Client, name types.NamespacedName, pod *corev1.Pod, userList []string) error {
	logf.Log.Info("Creating Approval...")
	label := GenerateUserLabel(userList)
	// Add Tekton labels, if exist
	for k, v := range pod.ObjectMeta.Labels {
		if strings.HasPrefix(k, TektonLabelPrefix) {
			label[k] = v
		}
	}
	approval := &tmaxv1.Approval{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    label,
		},
		Spec: tmaxv1.ApprovalSpec{
			PodName: pod.Name,
			Users:   userList,
		},
	}

	if err := c.Create(context.TODO(), approval); err != nil {
		return err
	}

	// Non-atomic status update... race may occur
	approvalWithStatus := approval.DeepCopy()
	approvalWithStatus.Status.Result = tmaxv1.ResultWaiting
	if err := c.Status().Patch(context.TODO(), approvalWithStatus, client.MergeFrom(approval)); err != nil {
		return err
	}
	return nil
}

func UpdateApproval(c client.Client, name types.NamespacedName, result tmaxv1.Result, reason string) error {
	logf.Log.Info("Updating Approval...")
	approval, err := GetApproval(c, name)
	if err != nil {
		return err
	}

	if approval.Status.Result == tmaxv1.ResultWaiting || approval.Status.Result == "" {
		approval.Status.Result = result
		approval.Status.Reason = reason
		approval.Status.DecisionTime = metav1.Now()
		if err := c.Status().Update(context.TODO(), approval); err != nil {
			return err
		}
	} else {
		logf.Log.Info(fmt.Sprintf("object Approval %s/%s is already in status %s", name.Namespace, name.Name, string(approval.Status.Result)))
	}

	return nil
}

func GetConfigMap(c client.Client, name types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := c.Get(context.TODO(), name, cm); err != nil {
		return nil, err
	}
	return cm, nil
}
