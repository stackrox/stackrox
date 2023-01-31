package util

import groovy.json.JsonSlurper
import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.client.LocalPortForward
import orchestratormanager.OrchestratorMain

import objects.Deployment
import objects.Service

@Slf4j
class MailServer {

    public static final Integer WEB_PORT = 1080
    public static final Integer SMTP_PORT = 1025
    public static final String MAILSERVER_USER = "user123"
    public static final String MAILSERVER_PASS = "soopersekret"

    @SuppressWarnings(["UnusedPrivateField"])
    private UUID uid
    private Deployment deployment
    private Service smtpSvc
    private Service webSvc
    private LocalPortForward webPortForward

    private MailServer() { }

    // Deploys a fake SMTP service that can both receive emails and return the emails it received
    static MailServer createMailServer(OrchestratorMain orchestrator,
                                       boolean authenticated = true,
                                       boolean useTLS = false) {
        def mailServer = new MailServer()
        mailServer.uid = UUID.randomUUID()
        def deploymentName = "maildev-${mailServer.uid}"
        try {
            def envVars = [
                    "MAILDEV_SMTP_PORT": SMTP_PORT.toString(),
                    "MAILDEV_WEB_PORT": WEB_PORT.toString(),
            ]

            if (authenticated) {
                envVars += [
                        "MAILDEV_INCOMING_USER": MAILSERVER_USER,
                        "MAILDEV_INCOMING_PASS": MAILSERVER_PASS,
                ]
            }

            mailServer.deployment =
                    new Deployment()
                            .setNamespace(orchestrator.getNameSpace())
                            .setName(deploymentName)
                            // The original is at docker.io/maildev/maildev:2.0.5
                            // and https://github.com/maildev/maildev
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:docker-io-maildev-maildev-2-0-5")
                            .addPort(WEB_PORT)
                            .addPort(SMTP_PORT)
                            .setEnv(envVars)
                            .addLabel("app", "maildev")
            orchestrator.createDeployment(mailServer.deployment)

            mailServer.smtpSvc = new Service("maildev-smtp-${mailServer.uid}", orchestrator.getNameSpace())
                    .addLabel("app", "maildev")
                    .addPort(SMTP_PORT, "TCP")
                    .setTargetPort(SMTP_PORT)
                    .setType(Service.Type.CLUSTERIP)
            orchestrator.createService(mailServer.smtpSvc)

            mailServer.webSvc = new Service("maildev-web-${mailServer.uid}", orchestrator.getNameSpace())
                    .addLabel("app", "maildev")
                    .addPort(WEB_PORT, "TCP")
                    .setTargetPort(WEB_PORT)
                    .setType(Service.Type.CLUSTERIP)
            orchestrator.createService(mailServer.webSvc)

            mailServer.webPortForward = orchestrator.
                    createPortForward(WEB_PORT, mailServer.deployment) as LocalPortForward
        } catch (Exception e) {
            log.info("Something bad happened, will run cleanup before failing", e)
            if (mailServer.smtpSvc) {
                orchestrator.deleteService(mailServer.smtpSvc.name, mailServer.smtpSvc.namespace)
            }
            if (mailServer.webSvc) {
                orchestrator.deleteService(mailServer.webSvc.name, mailServer.webSvc.namespace)
            }
            if (mailServer.deployment) {
                orchestrator.deleteDeployment(mailServer.deployment)
            }
            throw e
        }
        return mailServer
    }

    void teardown(OrchestratorMain orchestrator) {
        def imagePullSecrets = deployment.getImagePullSecret()
        for (String secret : imagePullSecrets) {
            orchestrator.deleteSecret(secret, deployment.namespace)
        }
        orchestrator.deleteService(smtpSvc.name, smtpSvc.namespace)
        orchestrator.deleteService(webSvc.name, webSvc.namespace)
        orchestrator.waitForServiceDeletion(smtpSvc)
        orchestrator.waitForServiceDeletion(webSvc)
        orchestrator.deleteDeployment(deployment)
    }

    String smtpUrl() {
        return "${smtpSvc.getName()}.${deployment.getNamespace()}:${SMTP_PORT}"
    }

    // Find all emails sent FROM the specified email address
    List findEmails(String fromEmail) {
        return fetchEmailsByUrl(String.format(
                "http://localhost:%s/email?from.address=%s", webPortForward.getLocalPort(),
                URLEncoder.encode(fromEmail, "UTF-8")))
    }

    // Find all emails sent TO the specified email address
    List findEmailsByToEmail(String toEmail) {
        return fetchEmailsByUrl(String.format(
                "http://localhost:%s/email?to.address=%s", webPortForward.getLocalPort(),
                URLEncoder.encode(toEmail, "UTF-8")))
    }

    // Download an attachment with the given file name for the specified email
    InputStream downloadEmailAttachment(String emailId, String fileName) {
        def url = String.format(
                "http://localhost:%s/email/%s/attachment/%s", webPortForward.getLocalPort(),
                URLEncoder.encode(emailId, "UTF-8"), URLEncoder.encode(fileName, "UTF-8"))

        def con = (HttpURLConnection) new URL(url).openConnection()
        con.setRequestMethod("GET")

        assert con.getResponseCode() == 200
        return con.getInputStream()
    }

    // Delete the specified email from the server
    def deleteEmail(String emailId) {
        def url = String.format(
                "http://localhost:%s/email/%s", webPortForward.getLocalPort(),
                URLEncoder.encode(emailId, "UTF-8"))

        def con = (HttpURLConnection) new URL(url).openConnection()
        con.setRequestMethod("DELETE")

        assert con.getResponseCode() == 200
    }

    private List fetchEmailsByUrl(String formattedUrl) {
        def con = (HttpURLConnection) new URL(formattedUrl).openConnection()
        con.setRequestMethod("GET")

        assert con.getResponseCode() == 200

        def jsonSlurper = new JsonSlurper()
        def objects = jsonSlurper.parseText(con.getInputStream().getText()) as List
        return objects
    }
}
