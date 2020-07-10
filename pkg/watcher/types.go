package watcher

import "github.com/tmax-cloud/approval-watcher/pkg/apis"

type Request struct {
	Decision apis.DecisionType `json:"decision"`
}

type Response struct {
	Result  bool   `json:"result"`
	Message string `json:"message,omitempty"`
}
