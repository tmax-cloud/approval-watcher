package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmax-cloud/approval-watcher/pkg/apis"
	"github.com/tmax-cloud/approval-watcher/pkg/watcher"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	tmaxv1 "github.com/tmax-cloud/approval-watcher/pkg/apis/tmax/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/tmax-cloud/approval-watcher/internal"
	"github.com/tmax-cloud/approval-watcher/internal/wrapper"
)

var reqMap sync.Map

func AddApproveApis(parent *wrapper.RouterWrapper) error {
	approveWrapper := wrapper.New("/approve", []string{http.MethodPut}, approveHandler)
	if err := parent.Add(approveWrapper); err != nil {
		return err
	}

	return nil
}

func AddRejectApis(parent *wrapper.RouterWrapper) error {
	approveWrapper := wrapper.New("/reject", []string{http.MethodPut}, rejectHandler)
	if err := parent.Add(approveWrapper); err != nil {
		return err
	}

	return nil
}

func approveHandler(w http.ResponseWriter, req *http.Request) {
	updateDecision(w, req, tmaxv1.ResultApproved)
}

func rejectHandler(w http.ResponseWriter, req *http.Request) {
	updateDecision(w, req, tmaxv1.ResultRejected)
}

func updateDecision(w http.ResponseWriter, req *http.Request, decision tmaxv1.Result) {
	userReq := &watcher.Request{}
	userResp := &watcher.Response{}

	for k, v := range req.Header {
		log.Info(fmt.Sprintf("HEADER : %s=%s", k, v))
	}

	// Check if there is auth field
	auth := req.Header.Get("Authorization")
	if auth == "" {
		_ = internal.RespondError(w, http.StatusUnauthorized, "authorization header should be given")
		return
	}

	// Get ns/approvalName
	vars := mux.Vars(req)

	ns, nsExist := vars["namespace"]
	approvalName, nameExist := vars["approvalName"]
	if !nsExist || !nameExist {
		_ = internal.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}

	// Get decision reason
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(userReq); err != nil {
		_ = internal.RespondError(w, http.StatusBadRequest, fmt.Sprintf("body is not in json form or is malformed, err : %s", err.Error()))
		return
	}

	// Get k8s client
	c, err := internal.Client(client.Options{})
	if err != nil {
		_ = internal.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get corresponding Approval object
	approval, err := internal.GetApproval(c, types.NamespacedName{Name: approvalName, Namespace: ns})
	if err != nil {
		_ = internal.RespondError(w, http.StatusBadRequest, fmt.Sprintf("no Approval %s/%s is found", ns, approvalName))
		return
	}

	// If Approval is already in approved/rejected status, respond with error
	if approval.Status.Result == tmaxv1.ResultApproved || approval.Status.Result == tmaxv1.ResultRejected {
		_ = internal.RespondError(w, http.StatusBadRequest, fmt.Sprintf("approval %s/%s is already in %s status", ns, approvalName, approval.Status.Result))
		return
	}

	// Get pod
	podName := approval.Spec.PodName
	pod := &corev1.Pod{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: ns}, pod); err != nil {
		_ = internal.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("no Pod %s/%s is found, err: %s", ns, podName, err.Error()))
		return
	}

	// Send request to pod
	podIP := pod.Status.PodIP
	sendMsg := apis.DecisionMessage{Decision: apis.DecisionType(decision)}

	sendClient := &http.Client{}
	jsonBody, err := json.Marshal(sendMsg)
	if err != nil {
		_ = internal.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("cannot marshal decisionmessage, err: %s", err.Error()))
		return
	}

	addr := fmt.Sprint("http://", podIP, ":", apis.StepServerPort, "/")
	sendReq, err := http.NewRequest(http.MethodPut, addr, bytes.NewBuffer(jsonBody))
	if err != nil {
		_ = internal.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("cannot create decision request, err: %s", err.Error()))
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
		_ = internal.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	defer sendResp.Body.Close()
	if sendResp.StatusCode == http.StatusOK {
		if err := internal.UpdateApproval(c, types.NamespacedName{Name: approval.Name, Namespace: approval.Namespace}, decision, userReq.Reason); err != nil {
			_ = internal.RespondError(w, http.StatusInternalServerError, err.Error())
		}
	} else {
		respObj := &watcher.Response{}
		dec := json.NewDecoder(sendResp.Body)
		if err := dec.Decode(respObj); err != nil {
			_ = internal.RespondError(w, http.StatusInternalServerError, err.Error())
		}
		_ = internal.RespondError(w, sendResp.StatusCode, respObj.Message)
		return
	}

	_ = internal.RespondJSON(w, userResp)
}
