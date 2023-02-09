package util

import groovy.util.logging.Slf4j
import objects.Deployment
import objects.Service
import orchestratormanager.OrchestratorMain

@Slf4j
class SyslogServer {
    public static final Integer SYSLOG_PORT = 514
    private Service syslogSvc
    private Deployment deployment

    static SyslogServer createRsyslog(OrchestratorMain orchestrator, String namespace) {
        def deploymentName = "syslog-${UUID.randomUUID()}"
        def rsyslog = new SyslogServer()
        try {
            rsyslog.deployment =
                    new Deployment()
                            .setNamespace(namespace)
                            .setName(deploymentName)
                            // The original is at docker.io/rsyslog/syslog_appliance_alpine:8.36.0-3.7
                            // and https://github.com/rsyslog/rsyslog-docker
                            .setImage("quay.io/rhacs-eng/qa:rsyslog_appliance_alpine")
                            .addPort(SYSLOG_PORT)
                            .addLabel("app", deploymentName)
            orchestrator.createDeployment(rsyslog.deployment)

            rsyslog.syslogSvc = new Service("rsyslog-service", orchestrator.getNameSpace())
                                            .addLabel("app", deploymentName)
                                            .addPort(SYSLOG_PORT , "TCP")
                                            .setTargetPort(SYSLOG_PORT)
                                            .setType(Service.Type.CLUSTERIP)
            orchestrator.createService(rsyslog.syslogSvc)
        }   catch (Exception e) {
            log.info("error creating syslog deployment or service", e)
            tearDown(orchestrator)
            throw e
        }
        return rsyslog
    }

    void tearDown(OrchestratorMain orchestrator) {
        if (syslogSvc) {
            orchestrator.deleteService(syslogSvc.name, syslogSvc.namespace)
        }
        if (deployment) {
            orchestrator.deleteDeployment(deployment)
        }
    }
}
