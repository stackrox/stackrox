package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.EmptyOuterClass
import io.stackrox.proto.api.v1.ScopedAccessControlServiceGrpc
import io.stackrox.proto.api.v1.ScopedAccessControlServiceOuterClass
import io.stackrox.proto.storage.AuthzPlugin
import io.stackrox.proto.storage.HttpEndpoint

class SACService extends BaseService {
    static getSACService() {
        return ScopedAccessControlServiceGrpc.newBlockingStub(getChannel())
    }

    static addAuthPlugin() {
        return getSACService().configureAuthzPlugin(
                ScopedAccessControlServiceOuterClass.UpsertAuthzPluginConfigRequest.newBuilder()
                        .setConfig(AuthzPlugin.AuthzPluginConfig.newBuilder()
                                .setName("SR Test Auth Plugin")
                                .setEnabled(true)
                                .setEndpointConfig(HttpEndpoint.HTTPEndpointConfig.newBuilder()
                                        .setEndpoint("https://authorization-plugin:443/authorize")
                                        .setSkipTlsVerify(true)
                                )
                        ).build()
        )
    }

    static getAuthPluginConfig() {
        return getSACService().getAuthzPluginConfigs(EmptyOuterClass.Empty.newBuilder().build())
    }

    static deleteAuthPluginConfig(String pluginConfigID) {
        return getSACService().deleteAuthzPlugin(Common.ResourceByID.newBuilder()
                .setId(pluginConfigID)
                .build()
        )
    }
}
