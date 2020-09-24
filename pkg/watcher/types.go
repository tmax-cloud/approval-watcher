package watcher

import "github.com/tmax-cloud/approval-watcher/pkg/apis"

type Request struct {
	Decision apis.DecisionType `json:"decision,omitempty"`
	Reason   string            `json:"reason,omitempty"`
}

type Response struct {
	Result  bool   `json:"result"`
	Message string `json:"message,omitempty"`
}
