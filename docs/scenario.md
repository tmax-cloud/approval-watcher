# 시나리오 실행 가이드
## 시나리오
1. 개발자 개발 완료 - PipelineRun 실행 --> SonarQube 통한 코드 정적 분석
2. `1차 승인` SonarQube 내역 확인 후 소스/이미지 빌드 승인
3. `2차 승인` 테스트 환경(`approval-test` 네임스페이스)에 배포 승인
4. `3차 승인` 테스트 환경에서 테스트 완료 승인
5. `4차 승인` 실제 서비스 환경(`approval-op` 네임스페이스) 배포 승인
## 주의사항
- SonarQube / Mail-sender / Approval-watcher는 모두 같은 네임스페이스 (`approval-system`)에 설치
## 사전 작업
1. [Approval Watcher 설치](installation.md)
2. [Mail-sender 설치](https://github.com/cqbqdd11519/mail-notifier/blob/master/docs/installation.md)  
3. SonarQube 설치  
(`<NAMESPACE>` 부분은 모두 PipelineRun이 구동될 Namespace로 치환)
   ```yaml
   apiVersion: tmax.io/v1
   kind: Template
   metadata:
     name: sonarqube-template
     namespace: <NAMESPACE>
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
     namespace: <NAMESPACE>
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
   - SonarQube (`http://<hostIP>:<PORT>/projects/create?mode=manual`) 접속해 `apache-sample-approved` 프로젝트 생성

## 시나리오 실행
1. Namespace/ServiceAccount/Role/RoleBinding 생성  
  (`<NAMESPACE>` 부분은 모두 PipelineRun이 구동될 Namespace로 치환)
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
     namespace: approval-test
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
     namespace: <NAMESPACE>
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
   ```yaml
   apiVersion: tekton.dev/v1alpha1
   kind: ClusterTask
   metadata:
     name: sonar-scan
   spec:
     description: Sonar scan task
     params:
       - name: SONAR_URL
         description: Sonar Qube server URL
       - name: SONAR_TOKEN
         description: Token for sonar qube
       - name: SONAR_PROJECT_ID
         description: Project ID in sonar qube
     resources:
       inputs:
         - name: source
           type: git
     results:
       - name: project-id
         description: Project ID for sonar qube
       - name: sonar-webhook-result
         description: webhook result from sonarqube
       - name: sonar-webhook-key
         description: webhook key
     steps:
       - name: pre
         image: tmaxcloudck/sonar-client:0.0.1
         imagePullPolicy: Always
         command:
           - node
           - --unhandled-rejections=strict
           - /client/index.js
           - pre
         env:
           - name: SONAR_URL
             value: $(params.SONAR_URL)
           - name: SONAR_TOKEN
             value: $(params.SONAR_TOKEN)
           - name: SONAR_PROJECT_ID
             value: $(params.SONAR_PROJECT_ID)
           - name: SONAR_PROJECT_ID_FILE
             value: $(results.project-id.path)
           - name: SONAR_WEBHOOK_KEY_FILE
             value: $(results.sonar-webhook-key.path)
       - name: build-and-scan
         image: sonarsource/sonar-scanner-cli:4.4
         imagePullPolicy: Always
         script: |
           sonar-scanner -Dsonar.host.url=$(params.SONAR_URL) -Dsonar.login=$(params.SONAR_TOKEN) -Dsonar.projectKey=$(cat $(results.project-id.path))
         workingDir: /workspace/source
       - name: post
         image: tmaxcloudck/sonar-client:0.0.1
         imagePullPolicy: Always
         command:
           - node
           - --unhandled-rejections=strict
           - /client/index.js
           - post
         env:
           - name: SONAR_URL
             value: $(params.SONAR_URL)
           - name: SONAR_TOKEN
             value: $(params.SONAR_TOKEN)
           - name: SONAR_RESULT_FILE
             value: /webhook-result/result.json
           - name: SONAR_WEBHOOK_KEY_FILE
             value: $(results.sonar-webhook-key.path)
         volumeMounts:
           - name: webhook-result
             mountPath: /webhook-result
     sidecars:
       - name: webhook
         image: tmaxcloudck/sonar-client:0.0.1
         imagePullPolicy: Always
         command:
           - node
           - /client/index.js
           - webhook
         env:
           - name: SONAR_RESULT_FILE
             value: /webhook-result/result.json
           - name: SONAR_RESULT_DEST
             value: $(results.sonar-webhook-result.path)
         volumeMounts:
           - name: webhook-result
             mountPath: /webhook-result
     volumes:
       - name: webhook-result
         emptyDir: {}
   ```
   ```yaml
   apiVersion: tekton.dev/v1alpha1
   kind: ClusterTask
   metadata:
     name: s2i-with-approval
   spec:
     description: S2I Task
     params:
     - description: The location of the s2i builder image.
       name: BUILDER_IMAGE
     - default: .
       description: The location of the path to run s2i from.
       name: PATH_CONTEXT
     - name: REGISTRY_SECRET_NAME
       description: Docker registry secret (kubernetes.io/dockerconfigjson type)
       default: ''
     - default: 'false'
       description: Verify the TLS on the registry endpoint (for push/pull to a non-TLS registry)
       name: TLSVERIFY
     - name: LOGLEVEL
       description: Log level when running the S2I binary
       default: '0'
     - name: PACKAGE_SERVER_URL
       description: URL (including protocol, ip, port, and path) of private package server (e.g., devpi, pypi, verdaccio, ...)
       default: ''
     - name: CM_APPROVER_DEV
     - name: MAIL_TITLE
     - name: MAIL_CONTENT
     resources:
       inputs:
       - name: source
         type: git
       outputs:
       - name: image
         type: image
     results:
     - name: image-url
       description: Tag-updated image url
     - name: registry-cred
       description: Tag-updated image url
     steps:
       - name: send-mail
         image: tmaxcloudck/mail-sender-client:latest
         imagePullPolicy: Always
         env:
         - name: MAIL_SERVER
           value: http://mail-sender.approval-system:9999/
         - name: MAIL_FROM
           value: no-reply-tc@tmax.co.kr
         - name: MAIL_SUBJECT
           value: $(params.MAIL_TITLE)
         - name: MAIL_CONTENT
           value: $(params.MAIL_CONTENT)
         volumeMounts:
         - name: approver-list-dev
           mountPath: /tmp/config
       - name: approve-1
         image: tmaxcloudck/approval-step-server:latest
         imagePullPolicy: Always
         volumeMounts:
         - name: approver-list-dev
           mountPath: /tmp/config
       - name: update-image-url
         image: tmaxcloudck/cicd-util:1.0.1
         script: |
           #!/bin/bash
           GIT_DIR="/workspace/source"
           ORIGINAL_URL="$(outputs.resources.image.url)"
           TARGET_FILE="$(results.image-url.path)"
           [ $(echo $ORIGINAL_URL | awk -F '/' '{printf $NF}' | awk -F ':' '{printf "%d", split($0,a)}') -eq 1 ] && TAG=":"$(git --git-dir=$GIT_DIR/.git rev-parse --short HEAD)
           echo "$ORIGINAL_URL$TAG" | tee $TARGET_FILE
       - name: parse-registry-cred
         image: tmaxcloudck/cicd-util:1.0.1
         script: |
           #!/bin/bash
           FILENAME="$(results.registry-cred.path)"
           if [ "$(params.REGISTRY_SECRET_NAME)" != "" ]; then
               IMAGE_URL=$(cat $(results.image-url.path))
               URL_ARR=(${IMAGE_URL//\// })
               REGISTRY=${URL_ARR[0]}
               NAMESPACE=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)
               ENCODED_CFG=$(kubectl -n $NAMESPACE get secrets $(params.REGISTRY_SECRET_NAME) -o jsonpath='{.data.\.dockerconfigjson}')
               if [ $(( $? + 0 )) -eq 0 ]; then
                   CRED=$(echo $ENCODED_CFG | base64 -d | jq -r '.auths."'$REGISTRY'".auth' --raw-output -c)
                   if [ "$CRED" == "null" ]; then
                     CRED=""
                   fi
                   echo $CRED | tee $FILENAME
               else
                   touch $FILENAME
               fi
           else
               touch $FILENAME
           fi
       - name: generate
         image: quay.io/openshift-pipeline/s2i:nightly
         script: |
           #!/bin/sh
           set -ex
           FILENAME=s2i.env
           touch $FILENAME
           if [ "$(inputs.params.PACKAGE_SERVER_URL)" != "" ]; then
             case "$(inputs.params.BUILDER_IMAGE)" in
               *python*) echo "PIP_INDEX_URL=$(inputs.params.PACKAGE_SERVER_URL)" >> $FILENAME
                         echo "PIP_TRUSTED_HOST=*" >> $FILENAME ;;
               *django*) echo "PIP_INDEX_URL=$(inputs.params.PACKAGE_SERVER_URL)" >> $FILENAME
                         echo "PIP_TRUSTED_HOST=*" >> $FILENAME ;;
               *nodejs*) echo "NPM_CONFIG_REGISTRY=$(inputs.params.PACKAGE_SERVER_URL)" >> $FILENAME;;
               *tomcat*) echo "MVN_CENTRAL_URL=$(inputs.params.PACKAGE_SERVER_URL)" >> $FILENAME;;
               *wildfly*) echo "MVN_CENTRAL_URL=$(inputs.params.PACKAGE_SERVER_URL)" >> $FILENAME;;
               *jeus*) echo "MVN_CENTRAL_URL=$(inputs.params.PACKAGE_SERVER_URL)" >> $FILENAME;;
             esac
           fi
           /usr/local/bin/s2i \
           --loglevel=$(inputs.params.LOGLEVEL) \
           -E $FILENAME \
           build $(inputs.params.PATH_CONTEXT) $(inputs.params.BUILDER_IMAGE) \
           --as-dockerfile /gen-source/Dockerfile.gen
         volumeMounts:
           - mountPath: /gen-source
             name: gen-source
         workingdir: /workspace/source
       - name: build
         image: quay.io/buildah/stable
         script: |
           buildah \
           bud \
           --format \
           docker \
           --tls-verify=$(inputs.params.TLSVERIFY) \
           --storage-driver=vfs \
           --layers \
           -f \
           /gen-source/Dockerfile.gen \
           -t \
           $(cat $(results.image-url.path)) \
           .
         securityContext:
           privileged: true
         volumeMounts:
           - mountPath: /var/lib/containers
             name: varlibcontainers
           - mountPath: /gen-source
             name: gen-source
         workingdir: /gen-source
       - name: push
         image: quay.io/buildah/stable
         script: |
           #!/bin/bash
           IMAGE_URL=$(cat $(results.image-url.path))
           REG_CRED=$(cat $(results.registry-cred.path) | base64 -d)
           if [ "$REG_CRED" != "" ]; then
               CRED="--creds=$REG_CRED"
           fi
           buildah \
           push \
           --tls-verify=$(inputs.params.TLSVERIFY) \
           --storage-driver=vfs \
           $CRED \
           $IMAGE_URL \
           docker://$IMAGE_URL
         securityContext:
           privileged: true
         volumeMounts:
           - mountPath: /var/lib/containers
             name: varlibcontainers
     volumes:
       - emptyDir: {}
         name: varlibcontainers
       - emptyDir: {}
         name: gen-source
       - name: approver-list-dev
         configMap:
           name: $(params.CM_APPROVER_DEV)
   
   ```
   ```yaml
   apiVersion: tekton.dev/v1alpha1
   kind: ClusterTask
   metadata:
     name: third-task
   spec:
     params:
     - name: third-user-1
       description: configmap name which contains users name
     - name: third-user-2
       description: configmap name which contains users name
     - name: mail-title-1
       default: "[배포 승인 요청] App 배포 승인 요청"
     - name: mail-content-1
       default: |
               namespace 환경에 QA 배포 예정
               승인이 필요합니다.
     - name: mail-title-2
       default: "[배포 확인 요청] 테스트 후 승인 요청"
     - name: mail-content-2
       default: |
               namespace 환경에 QA 배포 완료
               테스트 후 승인이 필요합니다.
     - name: app-name
       description: Deployment name
     - name: image-url
       description: Updated image url:tag
       default: $(inputs.resources.image.url)
     - name: deploy-namespace
       description: namespace to deploy deployment
     - name: deploy-cfg-name
       description: Deployment configmap name
       default: ""
     - name: deploy-env-json
       description: Deployment environment variable in JSON object form
       default: "{}"
     resources:
       inputs:
       - name: image
         type: image
     steps:
     - name: email-1
       image: tmaxcloudck/mail-sender-client:latest
       env:
       - name: MAIL_SERVER
         value: http://mail-sender.approval-system:9999/
       - name: MAIL_FROM
         value: no-reply-tc@tmax.co.kr
       - name: MAIL_SUBJECT
         value: $(params.mail-title-1)
       - name: MAIL_CONTENT
         value: $(params.mail-content-1)
       volumeMounts:
       - name: approver-list-1
         mountPath: /tmp/config
     - name: approve-1
       image: tmaxcloudck/approval-step-server:latest
       imagePullPolicy: Always
       volumeMounts:
       - name: approver-list-1
         mountPath: /tmp/config
     - name: create-yaml
       image: tmaxcloudck/cicd-util:1.0.1
       imagePullPolicy: Always
       command:
       - "make-deployment"
       args:
       - $(params.app-name)
       - $(params.image-url)
       volumeMounts:
       - mountPath: /generate
         name: generate
       env:
       - name: CONFIGMAP_NAME
         value: $(params.deploy-cfg-name)
       - name: DEPLOY_ENV_JSON
         value: $(params.deploy-env-json)
     - name: run-kubectl
       image: tmaxcloudck/cicd-util:1.0.1
       command:
       - "kubectl"
       args:
       - apply
       - -f
       - /generate/deployment.yaml
       - -n
       - $(params.deploy-namespace)
       volumeMounts:
       - mountPath: /generate
         name: generate
     - name: email-2
       image: tmaxcloudck/mail-sender-client:latest
       env:
       - name: MAIL_SERVER
         value: http://mail-sender.approval-system:9999/
       - name: MAIL_FROM
         value: no-reply-tc@tmax.co.kr
       - name: MAIL_SUBJECT
         value: $(params.mail-title-2)
       - name: MAIL_CONTENT
         value: $(params.mail-content-2)
       volumeMounts:
       - name: approver-list-2
         mountPath: /tmp/config
     - name: approve-2
       image: tmaxcloudck/approval-step-server:latest
       imagePullPolicy: Always
       volumeMounts:
       - name: approver-list-2
         mountPath: /tmp/config
     volumes:
     - name: approver-list-1
       configmap:
         name: $(params.third-user-1)
     - name: approver-list-2
       configmap:
         name: $(params.third-user-2)
     - name: generate
       emptyDir: {}
   ```
   ```yaml
   apiVersion: tekton.dev/v1alpha1
   kind: ClusterTask
   metadata:
     name: forth-task
   spec:
     params:
     - name: forth-user-1
       description: configmap name which contains users name
     - name: mail-title-1
     - name: mail-content-1
     - name: app-name
       description: Deployment name
     - name: image-url
       description: Updated image url:tag
       default: $(inputs.resources.image.url)
     - name: deploy-namespace
       description: namespace to deploy deployment
     - name: deploy-cfg-name
       description: Deployment configmap name
       default: ""
     - name: deploy-env-json
       description: Deployment environment variable in JSON object form
       default: "{}"
     resources:
       inputs:
       - name: image
         type: image
     steps:
     - name: email-1
       image: tmaxcloudck/mail-sender-client:latest
       env:
       - name: MAIL_SERVER
         value: http://mail-sender.approval-system:9999/
       - name: MAIL_FROM
         value: no-reply-tc@tmax.co.kr
       - name: MAIL_SUBJECT
         value: $(params.mail-title-1)
       - name: MAIL_CONTENT
         value: $(params.mail-content-1)
       volumeMounts:
       - name: approver-list-1
         mountPath: /tmp/config
     - name: approve-1
       image: tmaxcloudck/approval-step-server:latest
       imagePullPolicy: Always
       volumeMounts:
       - name: approver-list-1
         mountPath: /tmp/config
     - name: create-yaml
       image: tmaxcloudck/cicd-util:1.0.1
       imagePullPolicy: Always
       command:
       - "make-deployment"
       args:
       - $(params.app-name)
       - $(params.image-url)
       volumeMounts:
       - mountPath: /generate
         name: generate
       env:
       - name: CONFIGMAP_NAME
         value: $(params.deploy-cfg-name)
       - name: DEPLOY_ENV_JSON
         value: $(params.deploy-env-json)
     - name: run-kubectl
       image: tmaxcloudck/cicd-util:1.0.1
       command:
       - "kubectl"
       args:
       - apply
       - -f
       - /generate/deployment.yaml
       - -n
       - $(params.deploy-namespace)
       volumeMounts:
       - mountPath: /generate
         name: generate
     volumes:
     - name: approver-list-1
       configmap:
         name: $(params.forth-user-1)
     - name: generate
       emptyDir: {}
   ```
3. 템플릿 생성
    ```yaml
    apiVersion: tmax.io/v1
    kind: Template
    metadata:
      name: apache-cicd-with-approval
      namespace: <NAMESPACE>
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
                - name: mail-title-1
                  value: "[배포 승인 요청] $(params.app-name) 배포 승인 요청"
                - name: mail-content-1
                  value: |
                    이미지 푸시 완료
    
                    $(params.TEST_NS)에 APP 배포를 위한 승인 필요
                - name: mail-title-2
                  value: "[배포 확인 요청] $(params.app-name) 테스트 후 승인 요청"
                - name: mail-content-2
                  value: |
                    $(params.TEST_NS)에 $(params.app-name) 배포 완료
    
                    테스트 후 승인 요청
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
                - name: mail-title-1
                  value: "[배포 승인 요청] $(params.app-name) 운영 환경 배포 승인 요청"
                - name: mail-content-1
                  value: |
                    테스트 환경 승인 완료
    
                    운영 환경 배포 승인 요청
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
: `<NAMESPACE>` 부분은 모두 PipelineRun이 구동될 Namespace로 치환  
: `<이미지 레지스트리 주소>` 부분은 이미지 주소 입력 (e.g., `172.22.11.2:30500/apache-sample`)  
: `<User>=<Email>` 부분은 각각 HyperCloud User Object 이름 및 이메일 주소 입력 (다수 입력 시 콤마(,)로 구분해 입력)  
(e.g., admin-tmax.co.kr=admin@tmax.co.kr,qa-tmax.co.kr=qa@tmax.co.kr)  
: `<SonarQube Token>` 부분은 SonarQube 토큰 입력
    ```yaml
    apiVersion: tmax.io/v1
    kind: TemplateInstance
    metadata:
      name: apache-cicd-with-approval-instance
      namespace: <NAMESPACE>
    spec:
      template:
        metadata:
          name: apache-cicd-with-approval
        parameters:
          - name: APP_NAME
            value: apache-sample
          - name: NAMESPACE
            value: <NAMESPACE>
          - name: NAMESPACE_TEST
            value: approval-test
          - name: NAMESPACE_OP
            value: approval-op
          - name: GIT_URL
            value: https://github.com/microsoft/project-html-website
          - name: GIT_REV
            value: master
          - name: IMAGE_URL
            value: <이미지 레지스트리 주소>
          - name: REGISTRY_SECRET_NAME
            value: ''
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
            value: http://<SONAR-NodeIP>:<SONAR-NodePort>/
          - name: SONAR_TOKEN
            value: <SonarQube Token>
          - name: SONAR_PROJECT_ID
            value: apache-sample-approved
          - name: APPROVER_LIST_DEV
            value: <User>=<Email>
          - name: APPROVER_LIST_QA
            value: <User>=<Email>
          - name: APPROVER_LIST_OP
            value: <User>=<Email>
    ```
