package util

import groovy.util.logging.Slf4j

import orchestratormanager.OrchestratorMain

@Slf4j
class CollectorUtil {
    static final RUNTIME_CONFIG_MAP_NAME = "collector-config"
    static final RUNTIME_CONFIG_MAP_KEY = "runtime_config.yaml"

    static enableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, true)
    }

    static disableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, false)
    }

    static private setExternalIps(OrchestratorMain orchestrator, boolean state) {
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
