package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/tmax-cloud/approval-watcher/internal"
	"github.com/tmax-cloud/approval-watcher/pkg/apis"
	"net/http"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

var log = logf.Log.WithName("step-server")
var users map[string]string

const (
	ConfigMapPath string = "/tmp/config/users"
	JwtKey        string = "Tmax-ProAuth"
	Port          int    = 10203

	ApprovedMessage string = "Approval accepted. Exit the server."
	RejectedMessage string = "Reject accepted. Exit the server."
	UnknownMessage  string = "Decision Unknown: "
)

func extractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

func validateToken(tokenString string) (int, error) {
	claim := &apis.JwtClaim{}
	token, err := jwt.ParseWithClaims(tokenString, claim, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})

	if err != nil {
		log.Error(err, fmt.Sprintf("skip an error, token: %s", token))
	}

	if _, ok := users[claim.Id]; !ok {
		return http.StatusBadRequest, errors.New("not an approver in the list")
	}

	return 200, nil
}

func messageHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	var m apis.DecisionMessage
	err := json.NewDecoder(r.Body).Decode(&m)
	if err != nil {
		return 0, err
	}

	exitCode := 0
	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")

	var msg string
	if m.Decision == apis.DecisionApproved {
		msg = ApprovedMessage
	} else if m.Decision == apis.DecisionRejected {
		msg = RejectedMessage
		exitCode = 1
	} else {
		log.Info("Message: " + UnknownMessage)
		resMsg := apis.DecisionMessage{Decision: apis.DecisionUnknown, Message: UnknownMessage + string(m.Decision)}
		err = enc.Encode(resMsg)
		if err != nil {
			return 0, err
		}
		return 0, errors.New("unknown Message")
	}

	// approved or rejected
	log.Info("Message: " + msg)
	resMsg := apis.DecisionMessage{Decision: m.Decision, Message: msg}
	err = enc.Encode(resMsg)
	if err != nil {
		return 0, err
	}

	return exitCode, nil
}

func decisionHandler(w http.ResponseWriter, r *http.Request) {
	tokenString := extractToken(r)
	if tokenString == "" {
		log.Info("no access token in the request header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	responseCode, err := validateToken(tokenString)
	if err != nil {
		log.Error(err, "error occurs: ")
		w.WriteHeader(responseCode)
		return
	}

	exitCode, err := messageHandler(w, r)
	if err != nil {
		log.Error(err, "error occurs: ")
		return
	}

	// exit the server
	go func() {
		time.Sleep(5 * time.Second)
		os.Exit(exitCode)
	}()
}

func main() {
	var err error
	users, err = internal.Users(ConfigMapPath)
	if err != nil {
		log.Error(err, "error occurs while getting users list")
		panic(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/", decisionHandler).Methods("PUT")

	http.Handle("/", router)
	err = http.ListenAndServe(fmt.Sprintf(":%d", Port), nil)
	if err != nil {
		panic(err.Error())
	}
}
