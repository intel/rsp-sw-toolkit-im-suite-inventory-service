rrpBuildGoCode {
    projectKey = 'inventory-service'
    testDependencies = ['postgres']
    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']
    testStepsInParallel = true
    buildImage = 'amr-registry.caas.intel.com/rrp/ci-go-build-image:1.12.0-alpine'
    dockerImageName = "rsp/${projectKey}"
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"
    customBuildScript = "./build.sh"
    protexProjectName = 'bb-inventory-service'


    infra = [
        stackName: 'RSP-Codepipeline-InventoryService'
    ]

    notify = [
        slack: [ success: '#ima-build-success', failure: '#ima-build-failed' ]
    ]
}
