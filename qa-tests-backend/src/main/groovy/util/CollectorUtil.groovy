package util

import groovy.util.logging.Slf4j

import orchestratormanager.OrchestratorMain

import java.net.HttpURLConnection
import java.net.URL

import com.google.protobuf.util.JsonFormat

import sensor.Collector

@Slf4j
class CollectorUtil {
    static final String RUNTIME_CONFIG_MAP_NAME = "collector-config"
    static final String RUNTIME_CONFIG_MAP_KEY = "runtime_config.yaml"

    static final String ENABLED_VALUE = "ENABLED"
    static final String DISABLED_VALUE = "DISABLED"

    static parseJsonToProtobuf(String json) {
        sensor.Collector.CollectorConfig.Builder builder = sensor.Collector.CollectorConfig.newBuilder()
        JsonFormat.parser().merge(json, builder)
        return builder.build()
    }

    static introspectionQuery(String collectorAddress, String endpoint) {
        String uri = "http://${collectorAddress}${endpoint}"
        HttpURLConnection connection = new URL(uri).openConnection()

        // this might be unneeded?
        connection.setRequestMethod("GET")

        try {
            connection.connect()

            if (connection.responseCode != HttpURLConnection.HTTP_OK) {
                throw new RuntimeException("IntrospectionQuery failed with ${connection.responseMessage}")
            }
            def jsonResponse = connection.inputStream.text
            return parseJsonToProtobuf(jsonResponse)
        } catch (Exception e) {
            throw new RuntimeException("Error making request: ${e.message}", e)
        } finally {
            connection.disconnect()
        }
    }

    static waitForConfigToHaveState(OrchestratorMain orchestrator, String state, int timeoutSeconds = 90, int port = 8080) {
        def portForwards = orchestrator.createCollectorPortForwards(port)

        log.info "Waiting for Collector Config propagation (${portForwards.size()} pods)"
        int intervalSeconds = 1
        int waitTime
        def startTime = System.currentTimeMillis()
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            // if a pod has the right config, remove it from the list
            // we need to check
            portForwards.removeAll {
                def config = introspectionQuery("127.0.0.1:${it.getLocalPort()}", "/state/runtime-config")

                if (config.networking.externalIps.enabled.name() == state) {
                    return true
                }

                return false
            }
            sleep intervalSeconds * 1000
        }

        return false
    }

    static enableExternalIps(OrchestratorMain orchestrator, int timeoutSeconds = 90) {
        setExternalIps(orchestrator, ENABLED_VALUE)
        waitForConfigToHaveState(orchestrator, ENABLED_VALUE, timeoutSeconds)
    }

    static disableExternalIps(OrchestratorMain orchestrator, int timeoutSeconds = 90) {
        setExternalIps(orchestrator, DISABLED_VALUE)
        waitForConfigToHaveState(orchestrator, DISABLED_VALUE, timeoutSeconds)
    }

    static deleteRuntimeConfig(OrchestratorMain orchestrator) {
        orchestrator.deleteConfigMap(RUNTIME_CONFIG_MAP_NAME, "stackrox")
    }

    static private setExternalIps(OrchestratorMain orchestrator, String state) {
        String runtimeConfig = """\
networking:
  externalIps:
    enabled: ${state}
"""
        Map<String, String> data = [
            (RUNTIME_CONFIG_MAP_KEY): runtimeConfig,
        ]

        orchestrator.createConfigMap(RUNTIME_CONFIG_MAP_NAME, data, "stackrox")
    }
}
