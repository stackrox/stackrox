import groovy.json.JsonOutput
import groovy.json.JsonSlurper
import org.apache.commons.codec.digest.DigestUtils

import util.Env
import util.MaskingPatternLayout

import spock.lang.Specification

import org.slf4j.LoggerFactory
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.AppenderBase;

public class TestLogAppender extends AppenderBase<ILoggingEvent> {
    private final StringBuilder logs = new StringBuilder();
    private MaskingPatternLayout layout;

    public void setLayout(MaskingPatternLayout layout) {
        this.layout = layout;
    }

    @Override
    protected void append(ILoggingEvent eventObject) {
        String message = layout != null ? layout.doLayout(eventObject) : eventObject.getFormattedMessage();
        logs.append(message).append("APPENDED");
    }

    public String getLogs() {
        return logs.toString();
    }
}

class MaskingLoggedSecretsTest extends Specification {

    def "GOOGLE_CREDENTIALS not logged"() {
        when:
        def logAppender = new TestLogAppender();
        
        // Get the existing MaskingPatternLayout from the STDOUT appender configuration
        Logger logger = (Logger) LoggerFactory.getLogger(Logger.ROOT_LOGGER_NAME);
        def stdoutAppender = logger.getAppender("STDOUT");
        def encoder = stdoutAppender.getEncoder();
        def maskingLayout = encoder.getLayout();
        
        logAppender.setLayout(maskingLayout);
        logAppender.start();
        
        //Logger logger = (Logger) LoggerFactory.getLogger('MaskingTest');
        logger.setLevel(ch.qos.logback.classic.Level.INFO)
        logger.addAppender(logAppender);

        def originalString = 'service_account: {\"type\":\"service_account\",\"project_id\":\"projectname\",\"private_key_id\":\"34f1111111111111111111111111111111111d0c\",\"private_key\":\"-----BEGIN PRIVATE KEY-----\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nstringtohideXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXX=\\n-----END PRIVATE KEY-----\\n\",\"client_email\":\"XXXXXXXXXXXXXXXXXXXXXXXXXXs@acs-san-stackroxci.iam.gserviceaccount.com\",\"client_id\":\"114111111111111111229\",\"auth_uri\":\"https://accounts.google.com/o/oauth2/auth\",\"token_uri\":\"https://oauth2.googleapis.com/token\",\"auth_provider_x509_cert_url\":\"https://www.googleapis.com/oauth2/v1/certs\",\"client_x509_cert_url\":\"https://www.googleapis.com/robot/v1/metadata/x509/XXXXXXXXXXXXXXXXXXXXXXXXXXs%40acs-san-stackroxci.iam.gserviceaccount.com\",\"universe_domain\":\"googleapis.com\"}';
        logger.warn(originalString);
        logger.warn('LUJFR0lOIFBSSVZBVEUgSmoresecretbase64string') // base64 "BEGIN PRIVATE KEY"
        logger.warn('LS1CRUdJTiBQUklWQVRFImoresecretbase64string') // base64 "-BEGIN PRIVATE KEY"
        logger.warn('LS0tQkVHSU4gUFJJVkFURmoresecretbase64string') // base64 "--BEGIN PRIVATE KEY"

        then:
        String logs = logAppender.getLogs();
        // Verify that the original private key content is NOT present (it should be masked)
        ! logs.contains("-----BEGIN PRIVATE KEY-----\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX");
        ! logs.contains("-----BEGIN PRIVATE KEY-----");
        ! logs.contains("stringtohideXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX");

        // Verify that base64 patterns are properly masked
        ! logs.contains("LUJFR0lOIFBSSVZBVEUgSmoresecretbase64string")
        ! logs.contains("LS1CRUdJTiBQUklWQVRFImoresecretbase64string") 
        ! logs.contains("LS0tQkVHSU4gUFJJVkFURmoresecretbase64string")
        
        //access_key_id: .*
        //access_access_key: .*

        // Verify that the masked content IS present
        logs =~ /-----\*+END PRIVATE KEY-----/;
    }

    //def "CheckPropertyFileInputValue > GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY_V2"() {
}
