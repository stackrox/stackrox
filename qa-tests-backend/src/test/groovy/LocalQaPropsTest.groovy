import groovy.json.JsonOutput
import groovy.json.JsonSlurper
import org.apache.commons.codec.digest.DigestUtils

import util.Env

import spock.lang.IgnoreIf
import spock.lang.Specification

@IgnoreIf({ Env.IN_CI })
// these tests are some sanity checks for the values stored in the PROPERTIES_FILE matched
// few predefined expectations, but unfortunately the values drift and that test does not run.
// these predefined values are probably out of date & the test doesn't verify any functionality
// and hence skipped
@IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
class LocalQaPropsTest extends Specification {

    def "CheckPropertyFileInputValue > GOOGLE_CREDENTIALS_GCR_SCANNER"() {
        // When using GOOGLE_CREDENTIALS_GCR_SCANNER in qa-test-settings.properties
        // this test can be used to validate the reconstituted json credentials key.
        // No claims are made regarding key validity or authorizations. Only the
        // validity of the json data and exact content match via sha256 is performed.
        when:
        def originalString = Env.mustGet('GOOGLE_CREDENTIALS_GCR_SCANNER')
        def slurper = new JsonSlurper()
        def rawData = slurper.parseText(originalString)
        def canonicalJson = JsonOutput.toJson(rawData)
        def canonicalJsonSha256 = DigestUtils.sha256Hex(canonicalJson)
        then:
        canonicalJsonSha256 == 'f75d8cf9ea0c7886293f689478daafe75126c719313b4366c02bd41d69bb05e5'
    }

    def "CheckPropertyFileInputValue > GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY"() {
        when:
        def originalString = Env.mustGet('GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY')
        def slurper = new JsonSlurper()
        def rawData = slurper.parseText(originalString)
        def canonicalJson = JsonOutput.toJson(rawData)
        def canonicalJsonSha256 = DigestUtils.sha256Hex(canonicalJson)
        then:
        canonicalJsonSha256 == '0b7e83cefd9a8462f1c413dc04da7ab4d2a9712ae2dd4cc01ec8a745103c4429'
    }
}
