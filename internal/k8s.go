package internal

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmaxv1 "github.com/tmax-cloud/approval-watcher/pkg/apis/tmax/v1"
)

func GetApproval(c client.Client, name types.NamespacedName) (*tmaxv1.Approval, error) {
	approval := &tmaxv1.Approval{}
	if err := c.Get(context.TODO(), name, approval); err != nil {
		return nil, err
	}

	return approval, nil
}

func CreateApproval(c client.Client, name types.NamespacedName, podName string, userList []string) error {
	log.Println("Creating Approval...")
	approval := &tmaxv1.Approval{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    GenerateUserLabel(userList),
		},
		Spec: tmaxv1.ApprovalSpec{
			PodName: podName,
			Users:   userList,
		},
		Status: tmaxv1.ApprovalStatus{
			Result: tmaxv1.ResultWaiting,
		},
	}

	if err := c.Create(context.TODO(), approval); err != nil {
		return err
	}
	return nil
}

func UpdateApproval(c client.Client, name types.NamespacedName, result tmaxv1.Result) error {
	log.Println("Updating Approval...")
	approval, err := GetApproval(c, name)
	if err != nil {
		return err
	}

	if approval.Status.Result == tmaxv1.ResultWaiting || approval.Status.Result == "" {
		approval.Status.Result = result
		approval.Status.DecisionTime = metav1.Now()
		if err := c.Status().Update(context.TODO(), approval); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("object Approval %s/%s is already in status %s", name.Namespace, name.Name, string(approval.Status.Result))
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
