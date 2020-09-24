package v1

import (
	"fmt"
	"github.com/tmax-cloud/approval-watcher/internal"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/tmax-cloud/approval-watcher/internal/wrapper"
)

const (
	ApiGroup     = "approval.tmax.io"
	ApiVersion   = "v1"
	ApprovalKind = "approvals"
)

var log = logf.Log.WithName("approve-apis")

func AddV1Apis(parent *wrapper.RouterWrapper) error {
	versionWrapper := wrapper.New(fmt.Sprintf("/%s/%s", ApiGroup, ApiVersion), nil, versionHandler)
	if err := parent.Add(versionWrapper); err != nil {
		return err
	}

	namespaceWrapper := wrapper.New("/namespaces/{namespace}", nil, nil)
	if err := versionWrapper.Add(namespaceWrapper); err != nil {
		return err
	}

	return AddApprovalApis(namespaceWrapper)
}

func versionHandler(w http.ResponseWriter, _ *http.Request) {
	apiResourceList := &metav1.APIResourceList{}
	apiResourceList.Kind = "APIResourceList"
	apiResourceList.GroupVersion = fmt.Sprintf("%s/%s", ApiGroup, ApiVersion)
	apiResourceList.APIVersion = ApiVersion

	apiResourceList.APIResources = []metav1.APIResource{
		{
			Name:       fmt.Sprintf("%s/approve", ApprovalKind),
			Namespaced: true,
		},
		{
			Name:       fmt.Sprintf("%s/reject", ApprovalKind),
			Namespaced: true,
		},
	}

	_ = internal.RespondJSON(w, apiResourceList)
}
