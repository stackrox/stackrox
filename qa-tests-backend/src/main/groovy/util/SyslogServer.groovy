package util

import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.client.LocalPortForward

import objects.Deployment
import objects.Service
import orchestratormanager.OrchestratorMain

@Slf4j
class SyslogServer {
    public static final Integer SYSLOG_PORT = 514
    public static final Integer REST_PORT = 8080
    private Service syslogSvc
    private Service restSvc
    private Deployment deployment
    private LocalPortForward syslogPortForward

    static SyslogServer createRsyslog(OrchestratorMain orchestrator, String namespace) {
        def deploymentName = "syslog-${UUID.randomUUID()}"
        def rsyslog = new SyslogServer()
        try {
            rsyslog.deployment =
                    new Deployment()
                            .setNamespace(namespace)
                            .setName(deploymentName)
                            // The source for this image is in qa-tests-backend/test-images/syslog
                            // Run make syslog-image from the main folder to build the image
                            .setImage("quay.io/rhacs-eng/qa:syslog_server_1_0")
                            .setCommand(["/syslog"])
                            .addPort(SYSLOG_PORT)
                            .addLabel("app", deploymentName)
            orchestrator.createDeployment(rsyslog.deployment)

            rsyslog.syslogSvc = new Service("rsyslog-service", orchestrator.getNameSpace())
                                            .addLabel("app", deploymentName)
                                            .addPort(SYSLOG_PORT , "TCP")
                                            .setTargetPort(SYSLOG_PORT)
                                            .setType(Service.Type.CLUSTERIP)

            rsyslog.restSvc = new Service("rest-service", orchestrator.getNameSpace())
                    .addLabel("app", deploymentName)
                    .addPort(REST_PORT , "TCP")
                    .setTargetPort(REST_PORT)
                    .setType(Service.Type.CLUSTERIP)

            orchestrator.createService(rsyslog.syslogSvc)
            orchestrator.createService(rsyslog.restSvc)
            rsyslog.syslogPortForward = orchestrator.
                    createPortForward(REST_PORT, rsyslog.deployment) as LocalPortForward
        }   catch (Exception e) {
            log.info("error creating syslog deployment or service", e)
            tearDown(orchestrator)
            throw e
        }
        return rsyslog
    }

    String fetchLastMsg() {
        def formattedUrl = String.format("http://localhost:%s", syslogPortForward.getLocalPort())
        def con = (HttpURLConnection) new URL(formattedUrl).openConnection()
        con.setRequestMethod("GET")
        assert con.getResponseCode() == 200
        def res = con.getInputStream().getText()
        return res
    }

    void tearDown(OrchestratorMain orchestrator) {
        if (syslogSvc) {
            orchestrator.deleteService(syslogSvc.name, syslogSvc.namespace)
        }
        if (restSvc) {
            orchestrator.deleteService(restSvc.name, restSvc.namespace)
        }
        if (deployment) {
            orchestrator.deleteDeployment(deployment)
        }
    }
}
