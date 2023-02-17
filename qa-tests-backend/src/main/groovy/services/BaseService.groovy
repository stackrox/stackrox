package services

import groovy.transform.EqualsAndHashCode
import io.grpc.CallOptions
import io.grpc.Channel
import io.grpc.ClientCall
import io.grpc.ClientInterceptor
import io.grpc.ClientInterceptors
import io.grpc.ManagedChannel
import io.grpc.Metadata
import io.grpc.MethodDescriptor
import io.grpc.netty.GrpcSslContexts
import io.grpc.netty.NegotiationType
import io.grpc.netty.NettyChannelBuilder
import io.netty.handler.ssl.SslContextBuilder
import io.netty.handler.ssl.util.InsecureTrustManagerFactory
import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.EmptyOuterClass
import util.Env
import util.Keys

class BaseService {

    static final BASIC_AUTH_USERNAME = Env.mustGetUsername()
    static final BASIC_AUTH_PASSWORD = Env.mustGetPassword()

    static final EMPTY = EmptyOuterClass.Empty.newBuilder().build()

    static ResourceByID getResourceByID(String id) {
        return ResourceByID.newBuilder().setId(id).build()
    }

    static useApiToken(String apiToken) {
        updateAuthConfig(useClientCert, new AuthInterceptor(apiToken))
    }

    static useBasicAuth() {
        updateAuthConfig(useClientCert, new AuthInterceptor(BASIC_AUTH_USERNAME, BASIC_AUTH_PASSWORD))
    }

    static useNoAuthorizationHeader() {
        updateAuthConfig(useClientCert, null)
    }

    static setUseClientCert(boolean use) {
        updateAuthConfig(use, authInterceptor)
    }

    private static updateAuthConfig(boolean newUseClientCert, ClientInterceptor newAuthInterceptor) {
        if (useClientCert == newUseClientCert && authInterceptor == newAuthInterceptor) {
            return
        }
        if (useClientCert != newUseClientCert) {
            if (transportChannel != null) {
                transportChannel.shutdownNow()
                transportChannel = null
                effectiveChannel = null
            }
        }
        if (authInterceptor != newAuthInterceptor) {
            effectiveChannel = null
        }

        useClientCert = newUseClientCert
        authInterceptor = newAuthInterceptor
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
        protected void checkedStart(ClientCall.Listener<RespT> responseListener, Metadata headers) throws Exception {
            headers.put(AUTHORIZATION, authHeaderContents)
            delegate().start(responseListener, headers)
        }
    }

    @EqualsAndHashCode(includeFields = true)
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

    static ManagedChannel transportChannel = null
    static ClientInterceptor authInterceptor = null
    static Channel effectiveChannel = null
    private static boolean useClientCert = false

    static initializeChannel() {
        if (transportChannel == null) {
            SslContextBuilder sslContextBuilder = GrpcSslContexts
                    .forClient()
                    .trustManager(InsecureTrustManagerFactory.INSTANCE)
            if (useClientCert) {
                sslContextBuilder = sslContextBuilder.keyManager(Keys.keyManagerFactory())
            }
            def sslContext = sslContextBuilder.build()

            transportChannel = NettyChannelBuilder
                    .forAddress(Env.mustGetHostname(), Env.mustGetPort())
                    .enableRetry()
                    .negotiationType(NegotiationType.TLS)
                    .sslContext(sslContext)
                    .build()
            effectiveChannel = null
        }

        if (authInterceptor == null) {
            effectiveChannel = transportChannel
        } else {
            effectiveChannel = ClientInterceptors.intercept(transportChannel, authInterceptor)
        }
    }

    static Channel getChannel() {
        if (effectiveChannel == null) {
            initializeChannel()
        }
        return effectiveChannel
    }
}
