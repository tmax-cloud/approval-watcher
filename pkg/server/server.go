package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/tmax-cloud/approval-watcher/internal"
	"github.com/tmax-cloud/approval-watcher/pkg/apis"
	tmaxv1 "github.com/tmax-cloud/approval-watcher/pkg/apis/tmax/v1"
	"github.com/tmax-cloud/approval-watcher/pkg/watcher"
)

const (
	Method      string = "PUT"
	DefaultPort int    = 10999
	DefaultPath string = "/approve/{namespace}/{approvalName}"
)

var log = logf.Log.WithName("approve-server")
var reqMap sync.Map

/*
	DEPRECATED! - Use extension-api-server (pkg/apiserver)
	LaunchServer is only reserved for backward-compatibility (should be removed someday)
*/
func LaunchServer(port int, path string, _ chan bool) {
	router := mux.NewRouter()

	log.Info(fmt.Sprintf("Handler set to %s (%s)", path, Method))
	router.HandleFunc(path, handler).Methods(Method)

	addr := fmt.Sprintf(":%d", port)
	log.Info(fmt.Sprintf("Server is running on %s", addr))
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Error(err, "cannot listen")
		os.Exit(1)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	req := &watcher.Request{}
	resp := &watcher.Response{}

	encoder := json.NewEncoder(w)
	decoder := json.NewDecoder(r.Body)

	w.Header().Set("Content-Type", "application/json")

	// Get variables from URL
	vars := mux.Vars(r)
	ns, nsExist := vars["namespace"]
	approvalName, nameExist := vars["approvalName"]
	if !nsExist || !nameExist {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("url should be in form of %s", DefaultPath))
		return
	}

	// Check if there is auth field
	auth := r.Header.Get("Authorization")
	if auth == "" {
		respondError(w, http.StatusUnauthorized, "authorization header should be given")
		return
	}

	// Get decision
	if err := decoder.Decode(req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("body should contain decision field, err: %s", err.Error()))
		return
	}

	// Get k8s client
	c, err := internal.Client(client.Options{})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get corresponding Approval object
	approval, err := internal.GetApproval(c, types.NamespacedName{Name: approvalName, Namespace: ns})
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("no Approval %s/%s is found", ns, approvalName))
		return
	}

	// If Approval is already in approved/rejected status, respond with error
	if approval.Status.Result == tmaxv1.ResultApproved || approval.Status.Result == tmaxv1.ResultRejected {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("approval %s/%s is already in %s status", ns, approvalName, approval.Status.Result))
		return
	}

	// Get pod
	podName := approval.Spec.PodName
	pod := &corev1.Pod{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: ns}, pod); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("no Pod %s/%s is found, err: %s", ns, podName, err.Error()))
		return
	}

	// Send request to pod
	podIP := pod.Status.PodIP
	sendMsg := apis.DecisionMessage{Decision: req.Decision}

	sendClient := &http.Client{}
	jsonBody, err := json.Marshal(sendMsg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("cannot marshal decisionmessage, err: %s", err.Error()))
		return
	}

	addr := fmt.Sprint("http://", podIP, ":", apis.StepServerPort, "/")
	sendReq, err := http.NewRequest(http.MethodPut, addr, bytes.NewBuffer(jsonBody))
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("cannot create decision request, err: %s", err.Error()))
		return
	}
	sendReq.Header.Set("Authorization", auth)

	// Check if there is any ongoing request
	_, exist := reqMap.LoadOrStore(approvalName, sendReq)
	defer reqMap.Delete(approvalName)
	if exist {
		w.WriteHeader(http.StatusAccepted)
		resp := &watcher.Response{
			Result:  true,
			Message: fmt.Sprintf("approval %s is still in approval/reject progress", approvalName),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(err, "cannot return response")
		}
		return
	}

	log.Info(fmt.Sprintf("Sending request to %s", addr))

	sendResp, err := sendClient.Do(sendReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	defer sendResp.Body.Close()
	if sendResp.StatusCode == http.StatusOK {
		if err := internal.UpdateApproval(c, types.NamespacedName{Name: approval.Name, Namespace: approval.Namespace}, tmaxv1.Result(req.Decision), req.Reason); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
	} else {
		respObj := &watcher.Response{}
		dec := json.NewDecoder(sendResp.Body)
		if err := dec.Decode(respObj); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		respondError(w, sendResp.StatusCode, respObj.Message)
		return
	}

	// Return success
	resp.Result = true
	if err := encoder.Encode(resp); err != nil {
		log.Error(err, "cannot return response")
	}
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	log.Error(fmt.Errorf(message), "error occurred")

	resp := &watcher.Response{
		Result:  false,
		Message: message,
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error(err, "cannot return response")
	}
}
