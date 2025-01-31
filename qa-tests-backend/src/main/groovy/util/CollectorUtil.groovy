package util

import groovy.util.logging.Slf4j

import orchestratormanager.OrchestratorMain

import java.net.HttpURLConnection
import java.net.URL

@Slf4j
class CollectorUtil {
    static final String RUNTIME_CONFIG_MAP_NAME = "collector-config"
    static final String RUNTIME_CONFIG_MAP_KEY = "runtime_config.yaml"

    static final String ENABLED_VALUE = "ENABLED"
    static final String DISABLED_VALUE = "DISABLED"


    static enableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, ENABLED_VALUE)
        def result = introspectionQuery("127.0.0.1", "/state/runtime-config")
        println new String(result)
    }

    static disableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, DISABLED_VALUE)
        def result = introspectionQuery("127.0.0.1", "/state/runtime-config")
        println new String(result)
    }

    static deleteRuntimeConfig(OrchestratorMain orchestrator) {
        orchestrator.deleteConfigMap(RUNTIME_CONFIG_MAP_NAME, "stackrox")
        def result = introspectionQuery("127.0.0.1", "/state/runtime-config")
        println new String(result)
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


    def introspectionQuery(String collectorIP, String endpoint) {
        def uri = "http://${collectorIP}:8080${endpoint}"
        def connection = new URL(uri).openConnection() as HttpURLConnection
    
        try {
            connection.requestMethod = "GET"
            connection.connect()
    
            if (connection.responseCode != HttpURLConnection.HTTP_OK) {
                throw new RuntimeException("IntrospectionQuery failed with ${connection.responseMessage}")
            }
    
            return connection.inputStream.bytes
        } catch (Exception e) {
            throw new RuntimeException("Error making request: ${e.message}", e)
        } finally {
            connection.disconnect()
        }
    }
    
   //// Example usage:
   //try {
   //    def result = introspectionQuery("127.0.0.1", "/some-endpoint")
   //    println new String(result)
   //} catch (Exception e) {
   //    println "Error: ${e.message}"
   //}

}
