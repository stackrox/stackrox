package util

import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate

class Cert {
    static X509Certificate loadBase64EncodedCert(String base64Cert) {
        return (X509Certificate) CertificateFactory.getInstance("X.509").generateCertificate(
            new ByteArrayInputStream(base64Cert.decodeBase64())
        )
    }
}

