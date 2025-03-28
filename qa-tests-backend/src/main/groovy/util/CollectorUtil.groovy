package util

import static util.Helpers.withRetry

import common.Constants

import groovy.util.logging.Slf4j
import groovy.transform.CompileStatic

import orchestratormanager.Kubernetes

import com.google.protobuf.util.JsonFormat

import sensor.Collector

@Slf4j
@CompileStatic
class CollectorUtil {
    private static final String RUNTIME_CONFIG_MAP_NAME = "collector-config"
    private static final String RUNTIME_CONFIG_MAP_KEY = "runtime_config.yaml"

    private static final String ENABLED_VALUE = "ENABLED"
    private static final String DISABLED_VALUE = "DISABLED"

    static enableExternalIps(Kubernetes orchestrator, int timeoutSeconds = 90) {
        setExternalIps(orchestrator, ENABLED_VALUE)
        waitForConfigToHaveState(orchestrator, ENABLED_VALUE, timeoutSeconds)
    }

    static disableExternalIps(Kubernetes orchestrator, int timeoutSeconds = 90) {
        setExternalIps(orchestrator, DISABLED_VALUE)
        waitForConfigToHaveState(orchestrator, DISABLED_VALUE, timeoutSeconds)
    }

    static deleteRuntimeConfig(Kubernetes orchestrator) {
        orchestrator.deleteConfigMap(RUNTIME_CONFIG_MAP_NAME, Constants.STACKROX_NAMESPACE)
        return true
    }

    private static parseJsonToProtobuf(String json) {
        Collector.CollectorConfig.Builder builder = Collector.CollectorConfig.newBuilder()
        JsonFormat.parser().merge(json, builder)
        return builder.build()
    }

    private static introspectionQuery(String collectorAddress, String endpoint) {
        String uri = "http://${collectorAddress}${endpoint}"
        URL url = new URL(uri)
        HttpURLConnection connection = (HttpURLConnection) url.openConnection()

        // this might be unneeded?
        connection.setRequestMethod("GET")
        connection.connect()

        if (connection.responseCode != HttpURLConnection.HTTP_OK) {
            throw new RuntimeException("IntrospectionQuery failed with ${connection.responseMessage}")
        }
        def jsonResponse = connection.inputStream.text
        return parseJsonToProtobuf(jsonResponse)
    }

    private static waitForConfigToHaveState(Kubernetes orchestrator, String state, int timeout = 90, int port = 8080) {
        def portForwards = orchestrator.createCollectorPortForwards(port)

        log.info "Waiting for Collector Config propagation (${portForwards.size()} pods)"
        int intervalSeconds = 1
        int waitTime = 0
        int nretry = timeout / intervalSeconds
        withRetry(nretry, intervalSeconds) {
            // if a pod has the right config, remove it from the list
            // we need to check
            portForwards.removeAll {
                def config = introspectionQuery("127.0.0.1:${it.getLocalPort()}", "/state/runtime-config")
                def configTyped = (Collector.CollectorConfig) config
                return configTyped.networking.externalIps.enabled.name() == state
            }
            waitTime += intervalSeconds
            assert portForwards.isEmpty()
        }

        def success = portForwards.isEmpty()
        if (success) {
            log.info "Waited for ${waitTime} seconds for Collector runtime configuration to be updated"
        } else {
            log.info "Waiting for Collector runtime configuration timed out after ${timeout} seconds"
        }

        // if we timed out, some collectors have not updated
        // the config, so return false
        return success
    }

    static private setExternalIps(Kubernetes orchestrator, String state) {
        String runtimeConfig = """|
                                  |networking:
                                  |  externalIps:
                                  |    enabled: ${state}
                                  |""".stripMargin()
        Map<String, String> data = [
            (RUNTIME_CONFIG_MAP_KEY): runtimeConfig,
        ]

        orchestrator.createConfigMap(RUNTIME_CONFIG_MAP_NAME, data, Constants.STACKROX_NAMESPACE)
    }
}
