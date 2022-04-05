import util.Env
import spock.lang.Specification
import org.apache.commons.codec.digest.DigestUtils
import groovy.json.JsonSlurper
import groovy.json.JsonOutput

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

    // Disabled based on nightly failure https://issues.redhat.com/browse/ROX-9993
    // Might be ok now that expected value is updated but this test serves no value
    // in nightly and I want to ensure that it doesn't fail and mask other issues
    // that we do want to catch right away.
    //def "CheckPropertyFileInputValue > GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY"() {
    //    when:
    //    def originalString = Env.mustGet('GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY')
    //    def slurper = new JsonSlurper()
    //    def rawData = slurper.parseText(originalString)
    //    def canonicalJson = JsonOutput.toJson(rawData)
    //    def canonicalJsonSha256 = DigestUtils.sha256Hex(canonicalJson)
    //    then:
    //    canonicalJsonSha256 == '0b7e83cefd9a8462f1c413dc04da7ab4d2a9712ae2dd4cc01ec8a745103c4429'
    //}
}
