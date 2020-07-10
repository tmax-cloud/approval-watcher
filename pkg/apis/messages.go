package apis

type DecisionType string

const (
	DecisionApproved DecisionType = "Approved"
	DecisionRejected DecisionType = "Rejected"
	DecisionUnknown  DecisionType = "Unknown"
)

type DecisionMessage struct {
	Decision DecisionType `json:"decision"`
	Message  string       `json:"message"`
}
