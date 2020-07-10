package watcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmax-cloud/approval-watcher/pkg/apis"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/tmax-cloud/approval-watcher/internal"
)

const (
	Method      string = "POST"
	DefaultPort int    = 10999
	DefaultPath string = "/approve/{namespace}/{approvalName}"
)

func LaunchServer(port int, path string, _ chan bool) {
	router := mux.NewRouter()

	log.Printf("Handler set to %s (%s)\n", path, Method)
	router.HandleFunc(path, handler).Methods(Method)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Server is running on %s\n", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	req := &Request{}
	resp := &Response{}

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
		respondError(w, http.StatusBadRequest, "body should contain decision field")
		return
	}

	// Get k8s client
	c, err := internal.Client(client.Options{})

	// Get corresponding Approval object
	approval, err := internal.GetApproval(c, types.NamespacedName{Name: approvalName, Namespace: ns})
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("no Approval %s/%s is found", ns, approvalName))
		return
	}

	// Get pod
	podName := approval.Spec.PodName
	pod := &corev1.Pod{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: ns}, pod); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("no Pod %s/%s is found", ns, podName))
		return
	}

	// Send request to pod
	podIP := pod.Status.PodIP
	sendMsg := apis.DecisionMessage{Decision: req.Decision}

	sendClient := &http.Client{}
	jsonBody, err := json.Marshal(sendMsg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("cannot marshal decisionmessage"))
		return
	}
	sendReq, err := http.NewRequest(http.MethodPut, fmt.Sprint("http://", podIP, ":", 10203, "/"), bytes.NewBuffer(jsonBody))
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("cannot create decision request"))
		return
	}
	_, err = sendClient.Do(sendReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return success
	resp.Result = true
	if err := encoder.Encode(resp); err != nil {
		log.Println(err)
	}
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	log.Println(message)

	resp := &Response{
		Result:  false,
		Message: message,
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Println(err)
	}
}
