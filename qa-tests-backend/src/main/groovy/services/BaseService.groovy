package services

import io.grpc.ManagedChannel
import io.grpc.netty.GrpcSslContexts
import io.grpc.netty.NegotiationType
import io.grpc.netty.NettyChannelBuilder
import io.netty.handler.ssl.SslContext
import io.netty.handler.ssl.util.InsecureTrustManagerFactory

class BaseService {

    static ManagedChannel channelInstance = null

    static initializeChannel() {
        SslContext sslContext = GrpcSslContexts
                .forClient()
                .trustManager(InsecureTrustManagerFactory.INSTANCE)
                .build()

        int port = Integer.parseInt(System.getenv("PORT"))

        channelInstance = NettyChannelBuilder
                        .forAddress(System.getenv("HOSTNAME"), port)
                        .negotiationType(NegotiationType.TLS)
                        .sslContext(sslContext)
                        .build()
    }

    static getChannel() {
        if (channelInstance == null) {
            initializeChannel()
        }
        return channelInstance
    }
}
