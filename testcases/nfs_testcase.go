package testcases

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/disaster-recovery-acceptance-tests/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type NFSTestCase struct {
	uniqueTestID string
	instanceName string
}

func NewNFSTestCases() *NFSTestCase {
	id := RandomStringNumber()
	name := fmt.Sprintf("service-instance-%s", id)
	return &NFSTestCase{uniqueTestID: id, instanceName: name}
}

func (tc *NFSTestCase) Name() string {
	return "cf-nfsbroker"
}

func (tc *NFSTestCase) CheckDeployment(config Config) {
	By("checking if the nfsbroker app is present")
	RunCommandAndRetry("cf api --skip-ssl-validation", 3, config.CloudFoundryConfig.ApiUrl)
	RunCommandAndRetry("cf auth", 3, config.CloudFoundryConfig.AdminUsername, config.CloudFoundryConfig.AdminPassword)
	RunCommandSuccessfullyWithFailureMessage(tc.Name()+" test case cannot be run: space nfs-broker-space is not present", "cf target -o system -s nfs-broker-space")
	RunCommandSuccessfullyWithFailureMessage(tc.Name()+" test case cannot be run: app nfs-broker is not present", "cf app nfs-broker")
}

func (tc *NFSTestCase) BeforeBackup(config Config) {
	By("checking the service name and plane name are provided")
	Expect(config.CloudFoundryConfig.NFSServiceName).NotTo(BeEmpty(), "required config NFS service name not set")
	Expect(config.CloudFoundryConfig.NFSPlanName).NotTo(BeEmpty(), "required config NFS plan name not set")

	By("creating an NFS service broker and service instance")
	RunCommandSuccessfully("cf api --skip-ssl-validation", config.CloudFoundryConfig.ApiUrl)
	RunCommandSuccessfully("cf login --skip-ssl-validation -a", config.CloudFoundryConfig.ApiUrl,
		"-u", config.CloudFoundryConfig.AdminUsername, "-p", config.CloudFoundryConfig.AdminPassword)
	orgName := "acceptance-test-org-" + tc.uniqueTestID
	spaceName := "acceptance-test-space-" + tc.uniqueTestID
	RunCommandSuccessfully("cf create-org " + orgName)
	RunCommandSuccessfully("cf create-space " + spaceName + " -o " + orgName)
	RunCommandSuccessfully("cf target -o " + orgName + " -s " + spaceName)
	RunCommandSuccessfully("cf push dratsApp --docker-image docker/httpd --no-start --random-route")

	if config.CloudFoundryConfig.NFSCreateServiceBroker {
		RunCommandSuccessfully("cf create-service-broker nfsbroker-drats-" + tc.uniqueTestID + " " +
			config.CloudFoundryConfig.NFSBrokerUser + " " + config.CloudFoundryConfig.NFSBrokerPassword + " " +
			config.CloudFoundryConfig.NFSBrokerUrl)
	}

	RunCommandSuccessfully("cf enable-service-access " + config.CloudFoundryConfig.NFSServiceName + " -o " + orgName)
	RunCommandSuccessfully("cf create-service " + config.CloudFoundryConfig.NFSServiceName + " " +
		config.CloudFoundryConfig.NFSPlanName + " " + tc.instanceName + " -c " +
		`'{"share":"someserver.someplace.com/someshare"}'`)
}

func (tc *NFSTestCase) AfterBackup(config Config) {
	By("deleting the NFS service instance after backup")
	RunCommandSuccessfully("cf delete-service " + tc.instanceName + " -f")
}

func (tc *NFSTestCase) AfterRestore(config Config) {
	By("re-binding the NFS service instance after restore")
	RunCommandSuccessfully("cf bind-service dratsApp " + tc.instanceName + ` -c '{"uid":5000,"gid":5000}'`)
}

func (tc *NFSTestCase) Cleanup(config Config) {
	By("nfs cleanup")
	RunCommandSuccessfully("cf delete-org -f acceptance-test-org-" + tc.uniqueTestID)

	if config.CloudFoundryConfig.NFSCreateServiceBroker {
		RunCommandSuccessfully("cf delete-service-broker -f nfsbroker-drats-" + tc.uniqueTestID)
	}
}
