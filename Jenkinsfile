currentBuild.description = 'Cause: version bump'
node(label: 'centos7_router_devserver') {
    
    library 'jenkins-pipeline-library@main'
    cleanWs()
    
    agent {
        dockerfile true;
    }

    String PROMETHEUS_BRANCH = env.BRANCH_NAME
    stage("kubernetes-operator-build") {
        job = build job: 'pubsubplus-prometheus-operator-build', propagate: true, parameters:
        [
            string(name: 'PROMETHEUS_BRANCH', value: PROMETHEUS_BRANCH),
        ]
    }
}
