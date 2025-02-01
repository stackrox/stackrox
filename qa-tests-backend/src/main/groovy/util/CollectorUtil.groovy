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
        def builder = sensor.Collector.CollectorConfig.getMethod("newBuilder").invoke(null)
        JsonFormat.parser().merge(json, builder)
        return builder.build()
    }

    static introspectionQuery(String collectorIP, String endpoint) {
        def uri = "http://${collectorIP}:8080${endpoint}"
        def connection = new URL(uri).openConnection() as HttpURLConnection
    
        try {
            connection.requestMethod = "GET"
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

    static waitForConfigToHaveState(String state, int timeoutSeconds = 90) {
        int intervalSeconds = 1
        int waitTime
        def startTime = System.currentTimeMillis()
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            def config = introspectionQuery("127.0.0.1", "/state/runtime-config")
            if (config.networking.externalIps.enabled.name() == state) {
                return true
            }
            sleep intervalSeconds * 1000
        }

        return false
    }

    static private setExternalIpsAndWait(OrchestratorMain orchestrator, String state) {
        setExternalIps(orchestrator, state)
        waitForConfigToHaveState(state)
    }

    static enableExternalIpsAndWait(OrchestratorMain orchestrator) {
        setExternalIpsAndWait(orchestrator, ENABLED_VALUE)
    }

    static disableExternalIpsAndWait(OrchestratorMain orchestrator) {
        setExternalIpsAndWait(orchestrator, DISABLED_VALUE)
    }

    static enableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, ENABLED_VALUE)
    }

    static disableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, DISABLED_VALUE)
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
