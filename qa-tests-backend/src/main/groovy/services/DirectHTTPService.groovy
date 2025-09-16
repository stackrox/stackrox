package services

import java.security.SecureRandom
import javax.net.ssl.HttpsURLConnection
import javax.net.ssl.SSLContext

import com.google.gson.Gson
import com.google.gson.JsonObject
import groovy.transform.CompileStatic
import io.netty.handler.ssl.util.InsecureTrustManagerFactory

import util.Env

@CompileStatic
class DirectHTTPService {

    static final BASIC_AUTH_USERNAME = Env.mustGetUsername()
    static final BASIC_AUTH_PASSWORD = Env.mustGetPassword()
    static final BASE_URL = "https://${Env.mustGetHostname()}:${Env.mustGetPort()}"

    static SSLContext sslContext
    static {
        sslContext = SSLContext.getInstance("SSL")
        sslContext.init(null, InsecureTrustManagerFactory.INSTANCE.getTrustManagers(), new SecureRandom())
    }

    static HttpsURLConnection post(String url, JsonObject data = null) {
        def con = (HttpsURLConnection) new URL("${BASE_URL}/${url}").openConnection()
        con.setSSLSocketFactory(sslContext.getSocketFactory())
        con.setRequestMethod("POST")
        String encoded = Base64.getEncoder().
            encodeToString(("${BASIC_AUTH_USERNAME}:${BASIC_AUTH_PASSWORD}").getBytes("UTF-8"))
        con.setRequestProperty("Authorization", "Basic ${encoded}")
        if (data != null) {
            con.setDoOutput(true)
            con.setRequestProperty("Content-Type", "application/json; charset=UTF-8")
            con.getOutputStream().write(new Gson().toJson(data).getBytes())
        }

        return con
    }
}
