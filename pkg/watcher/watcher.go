package watcher

import (
	"bufio"
	"fmt"
	"github.com/tmax-cloud/approval-watcher/internal"
	"github.com/tmax-cloud/approval-watcher/pkg/apis"
	tmaxv1 "github.com/tmax-cloud/approval-watcher/pkg/apis/tmax/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type PodState string

const (
	PodStateUnknown    PodState = "Unknown"
	PodStateWaiting    PodState = "Waiting"
	PodStateRunning    PodState = "Running"
	PodStateTerminated PodState = "Terminated"
)

const (
	LabelTektonTaskRun       string = "tekton.dev/taskRun"
	ApprovalStepNamePrefix   string = "step-approval-"
	ApproverVolumeNamePrefix string = "approver-list-"
	ConfigMapKey             string = "users"
)

var k8sClient client.Client

func WatchPods(_ chan bool) {
	cfg, err := config.GetConfig()
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	w, err := clientSet.CoreV1().Pods("").Watch(metav1.ListOptions{LabelSelector: LabelTektonTaskRun})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Started to watch pods with label %s ...\n", LabelTektonTaskRun)

	// Generate k8sClient for creating/updating approval cr
	s := scheme.Scheme
	if err := apis.AddToScheme(s); err != nil {
		log.Println(err)
		return
	}
	k8sClient, err = internal.Client(client.Options{Scheme: s})
	if err != nil {
		log.Println("cannot get k8s client")
		return
	}

	for event := range w.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			log.Println("object is not a Pod type")
		}
		if !containsApprovalStep(pod) {
			continue
		}
		switch event.Type {
		case watch.Added, watch.Modified:
			handlePodEvent(pod)
		case watch.Deleted:
			// Should we handle pod deletion event?
		}
	}
}

func handlePodEvent(pod *corev1.Pod) {
	for i, cont := range pod.Spec.Containers {
		status := getContainerStatus(pod, &cont)
		if status == nil || !strings.HasPrefix(cont.Name, ApprovalStepNamePrefix) {
			continue
		}

		state := getStepState(pod, &cont)

		if state == PodStateRunning {
			// Running state - check if it is launched now!
			// If it is the first step, or previous step is terminated, Approval step is running
			if i == 0 || getStepState(pod, &pod.Spec.Containers[i-1]) == PodStateTerminated {
				handleApprovalStepStarted(pod, &cont)
				return
			}
		} else if state == PodStateTerminated {
			// Terminated state - check if it is finished now!
			// If it is the last step, or next step is running, Approval step is terminated
			if i == len(pod.Spec.Containers)-1 || getStepState(pod, &pod.Spec.Containers[i+1]) == PodStateRunning || getStepState(pod, &pod.Spec.Containers[i+1]) == PodStateWaiting {
				handleApprovalStepFinished(pod, &cont)
				return
			}
		}
	}
}

// Executed when approval step is started
func handleApprovalStepStarted(pod *corev1.Pod, cont *corev1.Container) {
	log.Println("Approval step is started...")
	contStatus := getContainerStatus(pod, cont)
	if contStatus.State.Running == nil {
		log.Printf("approval step is in wrong state (expecting %s)\n", string(PodStateRunning))
		return
	}

	name, err := generateApprovalName(pod, cont)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = internal.GetApproval(k8sClient, name)
	if err != nil && errors.IsNotFound(err) {
		// Create approval if it does not exist
		cmName, err := getConfigMapName(pod, cont)
		if err != nil {
			log.Println(err)
			return
		}
		cm, err := internal.GetConfigMap(k8sClient, cmName)
		if err != nil {
			log.Println(err)
			return
		}
		var users []string
		usersString, exist := cm.Data[ConfigMapKey]
		if !exist {
			log.Printf("the ConfigMap should contain key %s\n", ConfigMapKey)
			return
		}
		//TODO - refactor USERS func
		scanner := bufio.NewScanner(strings.NewReader(usersString))
		for scanner.Scan() {
			user := strings.Split(scanner.Text(), "=")
			users = append(users, user[0])
		}
		if err := scanner.Err(); err != nil {
			log.Println(err)
			log.Printf("cannot process users list %s\n", usersString)
			return
		}
		if err := internal.CreateApproval(k8sClient, name, pod.Name, users); err != nil {
			log.Println(err)
			return
		}
	} else if err != nil {
		log.Println(err)
	}
}

// Executed when approval step is ended
func handleApprovalStepFinished(pod *corev1.Pod, cont *corev1.Container) {
	log.Println("Approval step is finished...")
	contStatus := getContainerStatus(pod, cont)
	if contStatus.State.Terminated == nil {
		log.Printf("approval step is in wrong state (expecting %s)\n", string(PodStateTerminated))
		return
	}

	exitCode := contStatus.State.Terminated.ExitCode

	var result tmaxv1.Result
	if exitCode == 0 {
		result = tmaxv1.ResultApproved
	} else {
		result = tmaxv1.ResultRejected
	}
	name, err := generateApprovalName(pod, cont)
	if err != nil {
		log.Println(err)
		return
	}
	if err := internal.UpdateApproval(k8sClient, name, result); err != nil {
		log.Println(err)
		return
	}
}

func containsApprovalStep(pod *corev1.Pod) bool {
	hasStep := false
	hasVolume := false

	// Check if needed step exist
	for _, s := range pod.Spec.Containers {
		if strings.HasPrefix(s.Name, ApprovalStepNamePrefix) {
			hasStep = true
		}
	}

	// Check if needed volumes exist
	for _, v := range pod.Spec.Volumes {
		if strings.HasPrefix(v.Name, ApproverVolumeNamePrefix) {
			hasVolume = true
		}
	}

	return hasStep && hasVolume
}

func getContainerStatus(pod *corev1.Pod, step *corev1.Container) *corev1.ContainerStatus {
	for _, c := range pod.Status.ContainerStatuses {
		if c.Name == step.Name {
			return &c
		}
	}
	return nil
}

func getStepState(pod *corev1.Pod, step *corev1.Container) PodState {
	status := getContainerStatus(pod, step)
	if status == nil {
		return PodStateUnknown
	}

	if status.State.Waiting != nil {
		return PodStateWaiting
	}

	if status.State.Running != nil {
		return PodStateRunning
	}

	if status.State.Terminated != nil {
		return PodStateTerminated
	}

	return PodStateUnknown
}

func getConfigMapName(pod *corev1.Pod, cont *corev1.Container) (types.NamespacedName, error) {
	volumeName := ""
	for _, volumeMount := range cont.VolumeMounts {
		if strings.HasPrefix(volumeMount.Name, ApproverVolumeNamePrefix) {
			volumeName = volumeMount.Name
			break
		}
	}
	if volumeName == "" {
		return types.NamespacedName{}, fmt.Errorf("no volume mount starting with %s is found", ApproverVolumeNamePrefix)
	}

	for _, volume := range pod.Spec.Volumes {
		if volume.Name == volumeName && volume.ConfigMap != nil {
			return types.NamespacedName{Name: volume.ConfigMap.Name, Namespace: pod.Namespace}, nil
		}
	}

	return types.NamespacedName{}, fmt.Errorf("no ConfigMap volume found")
}

func generateApprovalName(pod *corev1.Pod, step *corev1.Container) (types.NamespacedName, error) {
	podName := pod.Name
	stepName := step.Name
	if !strings.HasPrefix(stepName, ApprovalStepNamePrefix) {
		return types.NamespacedName{}, fmt.Errorf("step name %s does not start with %s", stepName, ApprovalStepNamePrefix)
	}

	numStr := stepName[len(ApprovalStepNamePrefix):]
	approvalName := podName[:len(podName)-len(numStr)-1] + "-" + numStr

	return types.NamespacedName{Name: approvalName, Namespace: pod.Namespace}, nil
}
