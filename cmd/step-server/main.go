package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/tmax-cloud/approval-watcher/internal"
	"github.com/tmax-cloud/approval-watcher/pkg/apis"
	"github.com/tmax-cloud/approval-watcher/pkg/watcher"
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
	Port                 = apis.StepServerPort

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

func validateToken(tokenString string) error {
	claim := &apis.JwtClaim{}
	_, _, err := new(jwt.Parser).ParseUnverified(tokenString, claim)
	if err != nil {
		return err
	}

	var id string
	if claim.Id != "" { // Non-keycloak id
		id = claim.Id
	} else if claim.KeyCloakId != "" { // Keycloak id
		id = claim.KeyCloakId
	} else {
		return fmt.Errorf("token is malformed, token: %s", tokenString)
	}

	if _, ok := users[id]; !ok {
		return errors.New("not an approver in the list")
	}

	return nil
}

func messageHandler(m apis.DecisionMessage) (int, watcher.Response, error) {
	exitCode := 0

	var msg string
	if m.Decision == apis.DecisionApproved {
		log.Info("approved message accepted")
		msg = ApprovedMessage
	} else if m.Decision == apis.DecisionRejected {
		log.Info("rejected message accepted")
		msg = RejectedMessage
		exitCode = 1
	} else {
		log.Info("Message: " + UnknownMessage)
		resMsg := watcher.Response{Result: false, Message: UnknownMessage + string(m.Decision)}
		return 0, resMsg, errors.New("unknown Message")
	}

	// approved or rejected
	log.Info("Message: " + msg)
	resMsg := watcher.Response{Result: true, Message: msg}

	return exitCode, resMsg, nil
}

func responseMessage(w http.ResponseWriter, respCode int, respMsg watcher.Response) {
	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(respCode)
	err := enc.Encode(respMsg)

	if err != nil {
		log.Error(err, "error occurs while encoding response message")
	}
}

func decisionHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("request comes in")

	tokenString := extractToken(r)
	if tokenString == "" {
		log.Info("no access token in the request header")
		responseMessage(w, http.StatusUnauthorized, watcher.Response{
			Result:  false,
			Message: "invalid token",
		})
		return
	}

	err := validateToken(tokenString)
	if err != nil {
		log.Error(err, "error occurs while validating token")
		responseMessage(w, http.StatusUnauthorized, watcher.Response{
			Result:  false,
			Message: "invalid token",
		})
		return
	}

	// get message
	var m apis.DecisionMessage
	err = json.NewDecoder(r.Body).Decode(&m)
	if err != nil {
		log.Error(err, "error occurs while decoding body")
		responseMessage(w, http.StatusInternalServerError, watcher.Response{
			Result:  false,
			Message: "internal server error",
		})
		return
	}

	exitCode, respMsg, err := messageHandler(m)
	// if get unknown messages
	if err != nil {
		log.Error(err, "received unknown message")
		responseMessage(w, http.StatusBadRequest, respMsg)
		return
	}

	// succeed to get approved or rejected message
	log.Info(fmt.Sprintf("succeed to get message: %s", m.Decision))
	responseMessage(w, http.StatusOK, respMsg)

	// exit the server
	go func() {
		log.Info(fmt.Sprintf("server will be shutdown with exitcode %d", exitCode))
		time.Sleep(5 * time.Second)
		os.Exit(exitCode)
	}()
}

func main() {
	logf.SetLogger(zap.Logger())

	var err error
	users, err = internal.Users(ConfigMapPath)
	if err != nil {
		log.Error(err, "error occurs while getting users list")
		panic(err)
	}

	log.Info("initializing server....")
	router := mux.NewRouter()
	router.HandleFunc("/", decisionHandler).Methods("PUT")

	http.Handle("/", router)
	err = http.ListenAndServe(fmt.Sprintf(":%d", Port), nil)
	if err != nil {
		panic(err.Error())
	}
}
