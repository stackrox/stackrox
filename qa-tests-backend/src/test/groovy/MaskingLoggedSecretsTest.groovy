import groovy.json.JsonOutput
import groovy.json.JsonSlurper
import org.apache.commons.codec.digest.DigestUtils

import util.Env
import util.MaskingPatternLayout

import spock.lang.Specification
import spock.lang.Unroll

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
    private TestLogAppender logAppender
    private Logger logger

    def setup() {
        logAppender = new TestLogAppender()
        logAppender.setLayout(LoggerFactory.getLogger(Logger.ROOT_LOGGER_NAME).getAppender("STDOUT").getEncoder().getLayout())
        logAppender.start()
        
        logger = (Logger) LoggerFactory.getLogger("MaskingTest")
        logger.setLevel(ch.qos.logback.classic.Level.INFO)
        logger.addAppender(logAppender)
    }

    def cleanup() {
        if (logAppender) {
            logAppender.stop()
        }
    }

    def "Private key content is masked in Google service account JSON"() {
        given:
        def serviceAccountJson = '''service_account: {"type":"service_account","project_id":"projectname","private_key_id":"34f1111111111111111111111111111111111d0c","private_key":"-----BEGIN PRIVATE KEY-----\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nstringtohideXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\nXXXXXXXXXXXXXXXXXXXXXXX=\\n-----END PRIVATE KEY-----\\n","client_email":"XXXXXXXXXXXXXXXXXXXXXXXXXXs@acs-san-stackroxci.iam.gserviceaccount.com","client_id":"114111111111111111229","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_x509_cert_url":"https://www.googleapis.com/robot/v1/metadata/x509/XXXXXXXXXXXXXXXXXXXXXXXXXXs%40acs-san-stackroxci.iam.gserviceaccount.com","universe_domain":"googleapis.com"}'''

        when:
        logger.warn(serviceAccountJson)

        then:
        String logs = logAppender.getLogs()
        ! logs.contains("-----BEGIN PRIVATE KEY-----\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
        ! logs.contains("-----BEGIN PRIVATE KEY-----")
        ! logs.contains("stringtohideXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
        logs =~ /-----\*+END PRIVATE KEY-----/
    }

    @Unroll
    def "Base64 versions of private keys are masked: #description"() {
        when:
        logger.warn(secretString)

        then:
        String logs = logAppender.getLogs()
        ! logs.contains(secretString)
        logs.contains("*")

        where:
        secretString                                  | description
        'LUJFR0lOIFBSSVZBVEUgSmoresecretbase64string' | "base64 'BEGIN PRIVATE KEY'"
        'LS1CRUdJTiBQUklWQVRFImoresecretbase64string' | "base64 '-BEGIN PRIVATE KEY'"
        'LS0tQkVHSU4gUFJJVkFURmoresecretbase64string' | "base64 '--BEGIN PRIVATE KEY'"
    }

    @Unroll
    def "AWS access keys are masked: #description"() {
        when:
        logger.warn(logMessage)

        then:
        String logs = logAppender.getLogs()
        ! logs.contains(secretValue)
        logs.contains(expectedMaskedPattern)

        where:
        logMessage                                                    | secretValue                                | expectedMaskedPattern    | description
        'access_key_id: AKIAIOSFODNN7EXAMPLE'                         | 'AKIAIOSFODNN7EXAMPLE'                     | 'access_key_id: ***'     | "labeled access key"
        'secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY' | 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY' | 'secret_access_key: ***' | "labeled secret key"
        'valueAKIAIOSFODNN8EXAMPLEend'                                | 'AKIAIOSFODNN8EXAMPLE'                     | 'value***'               | "unlabeled access key"
        'valuewJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEYend'            | 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY' | 'value***'               | "labeled secret key"
    }

    def "Multiple secret types are masked in a single log entry"() {
        given:
        def logWithMultipleSecrets = '''
            Configuration loaded:
            aws_access_key_id = AKIAIOSFODNN7EXAMPLE
            aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
            private_key = "-----BEGIN PRIVATE KEY-----\\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\\n-----END PRIVATE KEY-----\\n"
            base64_encoded = LUJFR0lOIFBSSVZBVEUgSmoresecretbase64string
        '''

        when:
        logger.warn(logWithMultipleSecrets)

        then:
        String logs = logAppender.getLogs()
        
        // Verify no secrets are leaked
        ! logs.contains("AKIAIOSFODNN7EXAMPLE")
        ! logs.contains("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
        ! logs.contains("LUJFR0lOIFBSSVZBVEUgSmoresecretbase64string")
        ! logs.contains("-----BEGIN PRIVATE KEY-----")
        
        // Verify all are masked
        logs.contains("aws_access_key_id = ***")
        logs.contains("aws_secret_access_key = ***")
        logs =~ /-----\*+END PRIVATE KEY-----/
        logs.contains("*******************************************") // masked base64
    }
}
