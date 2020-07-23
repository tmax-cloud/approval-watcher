# Installation Guide

## Components
- Approval **CRD**
- Approval-watcher server **Deployment/Service**
- **Role/RoleBinding/ServiceAccount** for watcher server

## Procedure
1. Create Approval CRD
    ```bash
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/deploy/crds/tmax.io_approvals_crd.yaml`
    ```
2. Create service account, role, rolebinding
    ```bash 
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/deploy/service_account.yaml  
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/deploy/role.yaml
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/deploy/role_binding.yaml
    ```
3. Create watcher server
    ```bash
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/deploy/service.yaml
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/deploy/proxy-server.yaml
    ```
