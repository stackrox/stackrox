import static util.Helpers.withRetry

import io.stackrox.proto.storage.NotifierOuterClass

import common.Constants
import objects.Deployment
import objects.EmailNotifier
import services.CollectionsService
import services.VulnReportService
import util.MailServer
import util.Env

import org.junit.Assume
import spock.lang.Shared
import spock.lang.Tag

@Tag("PZ")
class VulnReportingTest extends BaseSpecification {

    static final private String SECONDARY_NAMESPACE = "vulnreport-2nd-namespace"
    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName("struts-deployment")
                    .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:struts-app")
                    .addLabel("app", "struts-test"),
            new Deployment()
                    .setName("registry-deployment")
                    .setNamespace(SECONDARY_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:struts-app")
                    .addLabel("app", "registry-image-test")
            // Use these if you want to actually test what the value of the report CSV is
//            new Deployment()
//                    .setName("nginx-deployment")
//                    .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
//                    .setImage("quay.io/rhacs-eng/qa:nginx-1-9")
//                    .addLabel("app", "nginx-test"),
//            new Deployment()
//                    .setName("nginx-deployment")
//                    .setNamespace(SECONDARY_NAMESPACE)
//                    .setImage("quay.io/rhacs-eng/qa:nginx-1-9")
//                    .addLabel("app", "nginx-test"),
    ]

    @Shared
    private MailServer mailServer

    def setupSpec() {
        mailServer = MailServer.createMailServer(orchestrator, true, false)
        sleep 60 * 1000 // wait 60s for service to start

        orchestrator.ensureNamespaceExists(SECONDARY_NAMESPACE)
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        DEPLOYMENTS.each { Services.waitForDeployment(it) }
    }

    def cleanupSpec() {
        if (mailServer) {
            mailServer.teardown(orchestrator)
        }

        DEPLOYMENTS.each { orchestrator.deleteDeployment(it) }
        orchestrator.deleteNamespace(SECONDARY_NAMESPACE)
    }

    @IgnoreIf({ true }) // temporarily skipped until this is migrated to use V2 API
    @Tag("BAT")
    def "Verify vulnerability generated using a collection sends an email with a valid report attachment"() {
        given:
        "Central is using postgres"
        Assume.assumeTrue(isPostgresRun())

        and:
        "a an email notifier is configured"
        EmailNotifier notifier = new EmailNotifier("Vuln Reports Notifier",
                mailServer.smtpUrl(),
                true, true, NotifierOuterClass.Email.AuthMethod.DISABLED)
        notifier.createNotifier()
        assert notifier.id
        // debug info
        log.info "notifier.id    ==== " + notifier.id
        log.info "notifier       ==== " + notifier

        and:
        "a collection is created"
        def collection = CollectionsService.createCollection(["struts-deployment"],
                [Constants.ORCHESTRATOR_NAMESPACE])
        assert collection.id
        // debug info
        log.info "collection.id  ==== " + collection.id
        log.info "collection     ==== " + collection

        and:
        "a report is configured"
        def report = VulnReportService.createVulnReportConfig(collection.id, notifier.id)
        assert report.id
        // debug info
        log.info "report.id      ==== " + report.id
        log.info "report         ==== " + report

        when:
        "a report is generated"
        assert VulnReportService.runReport(report.id)

        then:
        "the email server should've gotten an email with the report"
        List emails = []
        withRetry(4, 3) {
            emails = mailServer.findEmailsByToEmail(Constants.EMAIL_NOTIFER_SENDER)
            assert emails.size() >= 1
        }

        def email = emails[0]
        def emailId = (String) email["id"]

        // debug info
        log.info "emailId        ==== " + emailId
        log.info "email[subject] ==== " + email["subject"]
        log.info "email[html]    ==== " + email["html"]
        assert emailId
        assert email["subject"] =~ /(StackRox|RHACS) Image Vulnerability Report for (\d+)-(.*)-(\d+)/
        assert email["html"] =~ /has found vulnerabilities/

        // Since this is a BAT test, keep it simple and only validate we got the attachment and it's not 0 bytes
        Object[] attachments = email["attachments"]
        assert attachments.size() >= 2 // First attachment is the logo image, 2nd is the report

        def csvAttachmentMetadata = attachments.find {
            it["fileName"] =~ /(StackRox|RHACS)_Vulnerability_Report_(\d+)_(.*)_(\d+).zip/
        }
        assert csvAttachmentMetadata["fileName"]
        assert csvAttachmentMetadata["length"] > 0

        cleanup:
        "Cleanup resources"
        if (report) {
            VulnReportService.deleteVulnReportConfig(report.id)
            log.info "[Cleanup] Deleted vulnerability report config"
        }
        if (collection) {
            CollectionsService.deleteCollection(collection.id)
            log.info "[Cleanup] Deleted collection"
        }
        if (notifier) {
            notifier.deleteNotifier()
            log.info "[Cleanup] Deleted email notifier"
        }
        if (email) {
            mailServer.deleteEmail(emailId)
            log.info "[Cleanup] Deleted email from mail server"
        }
    }
}
