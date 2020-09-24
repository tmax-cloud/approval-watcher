module github.com/tmax-cloud/approval-watcher

go 1.13

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gorilla/mux v1.7.4
	github.com/operator-framework/operator-sdk v0.17.1
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/tools v0.0.0-20200709181711-e327e1019dfe // indirect
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.17.3
	knative.dev/pkg v0.0.0-20200623024526-fb0320d9287e
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	k8s.io/client-go => k8s.io/client-go v0.17.4
)
