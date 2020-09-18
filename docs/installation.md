# Installation Guide

## Components
- Approval **CRD**
- Approval-watcher server **Deployment/Service**
- **Role/RoleBinding/ServiceAccount** for watcher server

## Procedure
1. Create Approval CRD
    ```bash
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/deploy/crds/tmax.io_approvals_crd.yaml
    ```
2. Create namespace, service account, role, rolebinding
   ```yaml
   apiVersion: v1
   kind: Namespace
   metadata:
     name: approval-system
   ```
   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: approval-watcher
     namespace: approval-system
   ```
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: approval-watcher
   rules:
   - apiGroups:
     - ""
     resources:
     - pods
     - configmaps
     verbs:
     - create
     - delete
     - get
     - list
     - patch
     - update
     - watch
   - apiGroups:
     - tmax.io
     resources:
     - '*'
     verbs:
     - create
     - delete
     - get
     - list
     - patch
     - update
     - watch
   ```
   ```yaml
   kind: ClusterRoleBinding
   apiVersion: rbac.authorization.k8s.io/v1
   metadata:
     name: approval-watcher
   subjects:
   - kind: ServiceAccount
     name: approval-watcher
     namespace: approval-system
   roleRef:
     kind: ClusterRole
     name: approval-watcher
     apiGroup: rbac.authorization.k8s.io
   ```
3. Create watcher server
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: approval-proxy-server
     namespace: approval-system
   spec:
     selector:
       app: approval-proxy-server
     ports:
       - port: 80
         targetPort: 10999
   ```
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: approval-proxy-server
     namespace: approval-system
   spec:
     selector:
       matchLabels:
         app: approval-proxy-server
     template:
       metadata:
         labels:
           app: approval-proxy-server
       spec:
         serviceAccount: approval-watcher
         containers:
           - name: proxy-server
             image: tmaxcloudck/approval-watcher:latest
             imagePullPolicy: Always
   ```
