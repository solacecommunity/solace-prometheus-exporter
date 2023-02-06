currentBuild.description = 'Cause: version bump'
node(label: 'centos7_router_devserver') {
    
    library 'jenkins-pipeline-library@main'
    cleanWs()
    
    agent {
        dockerfile true;
    }

    String PROMETHEUS_BRANCH = env.BRANCH_NAME
    stage("prometheus-exporter-build") {
        job = build job: 'prometheus-exporter-build', propagate: true, parameters:
        [
            string(name: 'PROMETHEUS_BRANCH', value: PROMETHEUS_BRANCH),
        ]
    }
}
