package services

import io.grpc.CallOptions
import io.grpc.Channel
import io.grpc.ClientCall
import io.grpc.ClientCall.Listener
import io.grpc.ClientInterceptor
import io.grpc.ClientInterceptors
import io.grpc.ManagedChannel
import io.grpc.Metadata
import io.grpc.MethodDescriptor
import io.grpc.netty.GrpcSslContexts
import io.grpc.netty.NegotiationType
import io.grpc.netty.NettyChannelBuilder
import io.netty.handler.ssl.SslContext
import io.netty.handler.ssl.util.InsecureTrustManagerFactory
import util.Env

import java.util.concurrent.TimeUnit

class BaseService {

    private static String apiToken = null
    private static Boolean tokenUpdated = false

    static useApiToken(String apiToken) {
        this.apiToken = apiToken
        tokenUpdated = true
    }

    static useBasicAuth() {
        apiToken = null
        tokenUpdated = true
    }

    private static class CallWithAuthorizationHeader<ReqT, RespT>
            extends ClientInterceptors.CheckedForwardingClientCall<ReqT, RespT> {

        private static final Metadata.Key<String> AUTHORIZATION =
                Metadata.Key.of("Authorization", Metadata.ASCII_STRING_MARSHALLER)

        private final String authHeaderContents

        CallWithAuthorizationHeader(ClientCall<ReqT, RespT> delegate, String authHeaderContents) {
            super(delegate)
            this.authHeaderContents = authHeaderContents
        }

        @Override
        protected void checkedStart(Listener<RespT> responseListener, Metadata headers) throws Exception {
            headers.put(AUTHORIZATION, authHeaderContents)
            delegate().start(responseListener, headers)
        }
    }

    private static class AuthInterceptor implements ClientInterceptor {
        private final String authHeaderContents

        AuthInterceptor(String username, String password) {
            authHeaderContents = "Basic " + Base64.getEncoder().encodeToString(
                    (username + ":" + password).getBytes("UTF-8"))
        }

        AuthInterceptor(String apiToken) {
            authHeaderContents = "Bearer " + apiToken
        }

        public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
                MethodDescriptor<ReqT, RespT> method, CallOptions callOptions, Channel next) {
            return new CallWithAuthorizationHeader<>(next.newCall(method, callOptions), authHeaderContents)
        }
    }

    private static List<ClientInterceptor> interceptors() {
        String username = System.getenv("ROX_USERNAME") ?: ""
        String password = System.getenv("ROX_PASSWORD") ?: ""
        def interceptors = new ArrayList<ClientInterceptor>()

        if (apiToken != null) {
            interceptors.add(new AuthInterceptor(apiToken))
        } else if (!username.empty && !password.empty) {
            interceptors.add(new AuthInterceptor(username, password))
        }

        return interceptors
    }

    static ManagedChannel channelInstance = null

    static initializeChannel() {
        SslContext sslContext = GrpcSslContexts
                .forClient()
                .trustManager(InsecureTrustManagerFactory.INSTANCE)
                .build()

        channelInstance = NettyChannelBuilder
                        .forAddress(Env.mustGetHostname(), Env.mustGetPort())
                        .negotiationType(NegotiationType.TLS)
                        .sslContext(sslContext)
                        .intercept(interceptors())
                        .build()
    }

    static getChannel() {
        if (channelInstance == null) {
            initializeChannel()
        } else if (tokenUpdated) {
            channelInstance.shutdownNow()
            try {
                channelInstance.awaitTermination(30, TimeUnit.SECONDS)
            } catch (InterruptedException ie) {
                println "Channel did not terminate within timeout...: ${ie}"
            }
            initializeChannel()
            tokenUpdated = false
        }
        return channelInstance
    }
}
