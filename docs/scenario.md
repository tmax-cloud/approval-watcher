# 시나리오 실행 가이드
(`<NAMESPACE>` 부분은 PipelineRun이 구동될 네임스페이스 이름 입력)
## 사전 작업
1. SonarQube 설치
   ```yaml
   apiVersion: tmax.io/v1
   kind: Template
   metadata:
     name: sonarqube-template
   shortDescription: "SonarQube Deployment"
   longDescription: "SonarQube Deployment"
   imageUrl: "https://upload.wikimedia.org/wikipedia/commons/e/e6/Sonarqube-48x200.png"
   provider: tmax
   tags:
   - sonarqube
   objects:
   - apiVersion: v1
     kind: Service
     metadata:
       name: ${APP_NAME}-service
       namespace: ${NAMESPACE}
       labels:
         app: ${APP_NAME}
     spec:
       type: ${SERVICE_TYPE}
       ports:
       - name: http
         port: 9000
       selector:
         app: ${APP_NAME}
   - apiVersion: v1
     kind: PersistentVolumeClaim
     metadata:
       name: ${APP_NAME}-pvc
       namespace: ${NAMESPACE}
       labels:
         app: ${APP_NAME}
     spec:
       storageClassName: csi-cephfs-sc
       accessModes:
       - ReadWriteMany
       resources:
         requests:
           storage: ${STORAGE}
   - apiVersion: apps/v1
     kind: Deployment
     metadata:
       name: ${APP_NAME}
       namespace: ${NAMESPACE}
       labels:
         app: ${APP_NAME}
     spec:
       selector:
         matchLabels:
           app: ${APP_NAME}
       strategy:
         type: Recreate
       template:
         metadata:
           labels:
             app: ${APP_NAME}
         spec:
           containers:
           - name: ${APP_NAME}
             image: sonarqube:latest
             ports:
             - name: http
               containerPort: 9000
             volumeMounts:
             - name: ${APP_NAME}-pv
               mountPath: /opt/sonarqube/data
               subPath: data
             - name: ${APP_NAME}-pv
               mountPath: /opt/sonarqube/logs
               subPath: logs
             - name: ${APP_NAME}-pv
               mountPath: /opt/sonarqube/extentions
               subPath: extensions
           volumes:
           - name: ${APP_NAME}-pv
             persistentVolumeClaim:
               claimName: ${APP_NAME}-pvc
   parameters:
   - name: APP_NAME
     displayName: AppName
     description: A SonarQube Deployment Name
     required: true
   - name: NAMESPACE
     displayName: Namespace
     description: Application namespace
     required: true
   - name: STORAGE
     displayName: Storage
     description: Storage size
     required: true
   - name: SERVICE_TYPE
     displayName: ServiceType
     description: Service Type (ClsuterIP/NodePort/LoadBalancer)
     required: true
   plans:
   - name: sonarqube-plan1
     description: "SonarQube Plan"
     metadata:
       bullets:
       - "SonarQube Deployment Plan"
       costs:
         amount: 100
         unit: $
     free: false
     bindable: true
     plan_updateable: false
     schemas:
       service_instance:
         create:
           parameters:
             APP_NAME: sonarqube-deploy
             STORAGE: 10Gi
   ```
   ```yaml
   apiVersion: tmax.io/v1
   kind: TemplateInstance
   metadata:
     name: sonarqube-template-instance
   spec:
     template:
       metadata:
         name: sonarqube-template
       parameters:
       - name: APP_NAME
         value: sonarqube-test-deploy
       - name: NAMESPACE
         value: <NAMESPACE>
       - name: STORAGE
         value: 10Gi
       - name: SERVICE_TYPE
         value: LoadBalancer
   ```
   - SonarQube가 설치된 Node IP 및 NodePort 확인
   ```bash
   kubectl -n <NAMESPACE> get pod -l 'app=sonarqube-test-deploy' -o jsonpath='{.items[].status.hostIP}'
   kubectl -n <NAMESPACE> get service sonarqube-test-deploy-service -o jsonpath='{.spec.ports[0].nodePort}'
   ```
   - SonarQube (`http://<hostIP>:<PORT>/account/security` / ID: admin / PW: admin) 접속해 새로운 Token 생성 및 저장
   
2. [Mail-sender 설치](https://github.com/cqbqdd11519/mail-notifier/blob/master/docs/installation.md)
3. [Approval Watcher 설치](installation.md)

## 시나리오 실행
1. Namespace/ServiceAccount/Role/RoleBinding 생성
   ```yaml
   apiVersion: v1
   kind: Namespace
   metadata:
     name: approval-test
   ---
   apiVersion: v1
   kind: Namespace
   metadata:
     name: approval-op
   ```
   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: approval-sa
     namespace: <NAMESPACE>
   ```
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: approval-role
   rules:
   - apiGroups:
     - apps
     resources:
     - deployments
     verbs:
     - get
     - list
     - watch
     - create
     - update
     - patch
     - delete
   - apiGroups:
     - ""
     resources:
     - configmaps
     - secrets
     verbs:
     - get
     - list
     - create
     - update
     - patch
     - delete
   ```
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: RoleBinding
   metadata:
     name: approval-rolebinding
     namespace: approval-op
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: ClusterRole
     name: approval-role
   subjects:
   - kind: ServiceAccount
     name: approval-sa
     namespace: <NAMESPACE>
   ---
   apiVersion: rbac.authorization.k8s.io/v1
   kind: RoleBinding
   metadata:
     name: approval-rolebinding
     namespace: approval-op
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: ClusterRole
     name: approval-role
   subjects:
   - kind: ServiceAccount
     name: approval-sa
     namespace: <NAMESPACE>
   ```
2. Task 생성
    ```bash
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/example/pipeline/1st-task-general.yaml
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/example/pipeline/2nd-task.yaml
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/example/pipeline/3rd-task.yaml
    kubectl apply --filename https://raw.githubusercontent.com/tmax-cloud/approval-watcher/master/example/pipeline/4th-task.yaml
    ```
3. 템플릿 생성
    ```yaml
    apiVersion: tmax.io/v1
    kind: Template
    metadata:
      name: apache-cicd-with-approval
    plans:
      - name: default
    parameters:
      - name: APP_NAME
        displayName: AppName
        description: Application name
        required: true
      - name: NAMESPACE
        displayName: Namespace
        description: Application namespace
        required: true
      - name: NAMESPACE_TEST
        displayName: NamespaceTest
        description: Application namespace (test env.)
        required: true 
      - name: NAMESPACE_OP
        displayName: NamespaceOperation
        description: Application namespace (operation env.)
        required: true
      - name: GIT_URL
        displayName: GitURL
        description: Git Repo. URL
        required: true
      - name: GIT_REV
        displayName: GitRev
        description: Git Revision
        required: true
      - name: IMAGE_URL
        displayName: ImageURL
        description: Output Image URL
        required: true
      - name: REGISTRY_SECRET_NAME
        displayName: RegistrySecret
        description: Secret for accessing image registry
        required: false
        value: ''
      - name: SERVICE_ACCOUNT_NAME
        displayName: serviceAccountName
        description: Service Account Name
        required: true
      - name: WAS_PORT
        displayName: wasPort
        description: WAS Port
        valueType: number
        required: true
      - name: SERVICE_TYPE
        displayName: ServiceType
        description: Service Type (ClsuterIP/NodePort/LoadBalancer)
        required: true
      - name: PACKAGE_SERVER_URL
        displayName: PackageServerUrl
        description: URL (including protocol, ip, port, and path) of private package server
          (e.g., devpi, pypi, verdaccio, ...)
        required: false
      - name: DEPLOY_ENV_JSON
        displayName: DeployEnvJson
        description: Deployment environment variable in JSON object form
        required: false
        value: '{}'
      - name: SONAR_URL
        displayName: SonarUrl
        description: Sonar URL
        required: true
      - name: SONAR_TOKEN
        displayName: SonarToken
        description: Sonar Token
        required: true
      - name: SONAR_PROJECT_ID
        displayName: SonarProjectId
        description: Sonar Project ID
        required: true
      - name: APPROVER_LIST_DEV
        displayName: ApproverListDev
        description: Approver list - developer
        required: true
      - name: APPROVER_LIST_QA
        displayName: ApproverListQa
        description: Approver list - QA
        required: true
      - name: APPROVER_LIST_OP
        displayName: ApproverListOp
        description: Approver list - operator
        required: true
    objects:
      - apiVersion: v1
        kind: Service
        metadata:
          name: ${APP_NAME}-service
          namespace: ${NAMESPACE_TEST}
          labels:
            app: ${APP_NAME}
        spec:
          type: ${SERVICE_TYPE}
          ports:
            - port: ${WAS_PORT}
          selector:
            app: ${APP_NAME}
            tier: was
      - apiVersion: v1
        kind: Service
        metadata:
          name: ${APP_NAME}-service
          namespace: ${NAMESPACE_OP}
          labels:
            app: ${APP_NAME}
        spec:
          type: ${SERVICE_TYPE}
          ports:
            - port: ${WAS_PORT}
          selector:
            app: ${APP_NAME}
            tier: was
      - apiVersion: v1
        kind: ConfigMap
        metadata:
          name: ${APP_NAME}-approver-dev
          namespace: ${NAMESPACE}
          labels:
            app: ${APP_NAME}
        data:
          users: |
            ${APPROVER_LIST_DEV}
      - apiVersion: v1
        kind: ConfigMap
        metadata:
          name: ${APP_NAME}-approver-qa
          namespace: ${NAMESPACE}
          labels:
            app: ${APP_NAME}
        data:
          users: |
            ${APPROVER_LIST_QA}
      - apiVersion: v1
        kind: ConfigMap
        metadata:
          name: ${APP_NAME}-approver-op
          namespace: ${NAMESPACE}
          labels:
            app: ${APP_NAME}
        data:
          users: |
            ${APPROVER_LIST_OP}
      - apiVersion: v1
        kind: ConfigMap
        metadata:
          name: ${APP_NAME}-deploy-cfg
          namespace: ${NAMESPACE}
          labels:
            app: ${APP_NAME}
        data:
          deploy-spec.yaml: |
            spec:
              selector:
                matchLabels:
                  app: ${APP_NAME}
                  tier: was
              template:
                metadata:
                  labels:
                    app: ${APP_NAME}
                    tier: was
                spec:
                  imagePullSecrets:
                  - name: ${REGISTRY_SECRET_NAME}
                  containers:
                  - ports:
                    - containerPort: ${WAS_PORT}
      - apiVersion: tekton.dev/v1alpha1
        kind: PipelineResource
        metadata:
          name: ${APP_NAME}-input-git
          namespace: ${NAMESPACE}
          labels:
            app: ${APP_NAME}
        spec:
          type: git
          params:
            - name: revision
              value: ${GIT_REV}
            - name: url
              value: ${GIT_URL}
      - apiVersion: tekton.dev/v1alpha1
        kind: PipelineResource
        metadata:
          name: ${APP_NAME}-output-image
          namespace: ${NAMESPACE}
          labels:
            app: ${APP_NAME}
        spec:
          type: image
          params:
            - name: url
              value: ${IMAGE_URL}
      - apiVersion: tekton.dev/v1alpha1
        kind: Pipeline
        metadata:
          name: ${APP_NAME}-pipeline
          namespace: ${NAMESPACE}
          labels:
            app: ${APP_NAME}
        spec:
          params:
            - description: Application name
              name: app-name
              type: string
            - description: Configmap name for description
              name: deploy-cfg-name
              type: string
            - description: Deployment environment variable in JSON object form
              name: deploy-env-json
              type: string
            - name: TEST_NS
            - name: OP_NS
            - name: SONAR_URL
            - name: SONAR_TOKEN
            - name: SONAR_PROJECT_ID
            - name: CM_APPROVER_DEV
            - name: CM_APPROVER_QA
            - name: CM_APPROVER_OP
          resources:
            - name: source-repo
              type: git
            - name: image
              type: image
          tasks:
            - name: analyze
              taskRef:
                name: sonar-scan
                kind: ClusterTask
              resources:
                inputs:
                  - name: source
                    resource: source-repo
              params:
                - name: SONAR_URL
                  value: $(params.SONAR_URL)
                - name: SONAR_TOKEN
                  value: $(params.SONAR_TOKEN)
                - name: SONAR_PROJECT_ID
                  value: $(params.SONAR_PROJECT_ID)
            - name: build-source
              taskRef:
                name: s2i-with-approval
                kind: ClusterTask
              runAfter:
                - analyze
              resources:
                inputs:
                  - name: source
                    resource: source-repo
                outputs:
                  - name: image
                    resource: image
              params:
                - name: BUILDER_IMAGE
                  value: tmaxcloudck/s2i-apache:2.4
                - name: PACKAGE_SERVER_URL
                  value: ${PACKAGE_SERVER_URL}
                - name: REGISTRY_SECRET_NAME
                  value: ${REGISTRY_SECRET_NAME}
                - name: CM_APPROVER_DEV
                  value: $(params.CM_APPROVER_DEV)
                - name: MAIL_TITLE
                  value: "[빌드 승인 요청] $(params.app-name) 승인 요청"
                - name: MAIL_CONTENT
                  value: |
                    소스 코드 분석 완료
    
                    $(params.SONAR_URL)/dashboard?id=$(params.SONAR_PROJECT_ID) (접속정보: admin/admin)
    
                    결과 확인 후 빌드 승인 요청
            - name: deploy-test
              taskRef:
                kind: ClusterTask
                name: third-task
              runAfter:
                - build-source
              resources:
                inputs:
                  - name: image
                    resource: image
              params:
                - name: app-name
                  value: $(params.app-name)
                - name: image-url
                  value: $(tasks.build-source.results.image-url)
                - name: deploy-cfg-name
                  value: $(params.deploy-cfg-name)
                - name: deploy-env-json
                  value: $(params.deploy-env-json)
                - name: deploy-namespace
                  value: $(params.TEST_NS)
                - name: third-user-1
                  value: $(params.CM_APPROVER_QA)
                - name: third-user-2
                  value: $(params.CM_APPROVER_QA)
            - name: deploy-op
              taskRef:
                kind: ClusterTask
                name: forth-task
              runAfter:
                - deploy-test
              resources:
                inputs:
                  - name: image
                    resource: image
              params:
                - name: app-name
                  value: $(params.app-name)
                - name: image-url
                  value: $(tasks.build-source.results.image-url)
                - name: deploy-cfg-name
                  value: $(params.deploy-cfg-name)
                - name: deploy-env-json
                  value: $(params.deploy-env-json)
                - name: deploy-namespace
                  value: $(params.OP_NS)
                - name: forth-user-1
                  value: $(params.CM_APPROVER_QA)
      - apiVersion: tekton.dev/v1alpha1
        kind: PipelineRun
        metadata:
          name: ${APP_NAME}-pipeline-run
        spec:
          pipelineRef:
            name: ${APP_NAME}-pipeline
          serviceAccountName: ${SERVICE_ACCOUNT_NAME}
          resources:
          - name: source-repo
            resourceRef:
              name: ${APP_NAME}-input-git
          - name: image
            resourceRef:
              name: ${APP_NAME}-output-image
          params:
            - name: app-name
              value: ${APP_NAME}
            - name: deploy-cfg-name
              value: ${APP_NAME}-deploy-cfg
            - name: deploy-env-json
              value: ${DEPLOY_ENV_JSON}
            - name: TEST_NS
              value: ${NAMESPACE_TEST}
            - name: OP_NS
              value: ${NAMESPACE_OP}
            - name: SONAR_URL
              value: ${SONAR_URL}
            - name: SONAR_TOKEN
              value: ${SONAR_TOKEN}
            - name: SONAR_PROJECT_ID
              value: ${SONAR_PROJECT_ID}
            - name: CM_APPROVER_DEV
              value: ${APP_NAME}-approver-dev
            - name: CM_APPROVER_QA
              value: ${APP_NAME}-approver-qa
            - name: CM_APPROVER_OP
              value: ${APP_NAME}-approver-op
          timeout: 120h
    ```

4. 템플릿 인스턴스 생성
    ```yaml
    apiVersion: tmax.io/v1
    kind: TemplateInstance
    metadata:
      name: apache-cicd-with-approval-instance
    spec:
      template:
        metadata:
          name: apache-cicd-with-approval
        parameters:
          - name: APP_NAME
            value: apache-sample
          - name: NAMESPACE
            value: ck2-2
          - name: NAMESPACE_TEST
            value: approval-test
          - name: NAMESPACE_OP
            value: approval-op
          - name: GIT_URL
            value: https://github.com/microsoft/project-html-website
          - name: GIT_REV
            value: master
          - name: IMAGE_URL
            value: 192.168.6.224:443/apache-approved-sample
          - name: REGISTRY_SECRET_NAME
            value: hpcd-registry-cicd-test
          - name: SERVICE_ACCOUNT_NAME
            value: approval-sa
          - name: WAS_PORT
            value: 8080
          - name: SERVICE_TYPE
            value: NodePort
          - name: PACKAGE_SERVER_URL
            value: ''
          - name: DEPLOY_ENV_JSON
            value: '{}'
          - name: SONAR_URL
            value: http://192.168.6.200:32366/
          - name: SONAR_TOKEN
            value: 61eaa750227bcf5dbceadd2646bb7686f7e1c65a
          - name: SONAR_PROJECT_ID
            value: apache-sample-approved
          - name: APPROVER_LIST_DEV
            value: shkim-tmax.co.kr=sunghyun_kim3@tmax.co.kr
          - name: APPROVER_LIST_QA
            value: shkim-tmax.co.kr=sunghyun_kim3@tmax.co.kr
          - name: APPROVER_LIST_OP
            value: shkim-tmax.co.kr=sunghyun_kim3@tmax.co.kr
    ```
