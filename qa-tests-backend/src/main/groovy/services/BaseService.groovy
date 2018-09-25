package services

import io.grpc.netty.GrpcSslContexts
import io.grpc.netty.NegotiationType
import io.grpc.netty.NettyChannelBuilder
import io.netty.handler.ssl.SslContext
import io.netty.handler.ssl.util.InsecureTrustManagerFactory

class BaseService {

    static getChannel() {
        SslContext sslContext = GrpcSslContexts
                .forClient()
                .trustManager(InsecureTrustManagerFactory.INSTANCE)
                .build()

        int port = Integer.parseInt(System.getenv("PORT"))

        def channel = NettyChannelBuilder
                .forAddress(System.getenv("HOSTNAME"), port)
                .negotiationType(NegotiationType.TLS)
                .sslContext(sslContext)
                .build()
        return channel
    }

}
