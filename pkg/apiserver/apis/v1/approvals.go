package v1

import (
	"fmt"

	"github.com/tmax-cloud/approval-watcher/internal/wrapper"
)

func AddApprovalApis(parent *wrapper.RouterWrapper) error {
	approvalWrapper := wrapper.New(fmt.Sprintf("/%s/{approvalName}", ApprovalKind), nil, nil)
	if err := parent.Add(approvalWrapper); err != nil {
		return err
	}

	approvalWrapper.Router.Use(Authorize)

	if err := AddApproveApis(approvalWrapper); err != nil {
		return err
	}
	if err := AddRejectApis(approvalWrapper); err != nil {
		return err
	}
	return nil
}
