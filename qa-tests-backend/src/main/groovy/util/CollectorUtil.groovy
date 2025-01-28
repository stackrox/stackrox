package util

import groovy.util.logging.Slf4j

import orchestratormanager.OrchestratorMain

@Slf4j
class CollectorUtil {
    static final String RUNTIME_CONFIG_MAP_NAME = "collector-config"
    static final String RUNTIME_CONFIG_MAP_KEY = "runtime_config.yaml"

    static final String ENABLED_KEY = "ENABLED"
    static final String DISABLED_KEY = "DISABLED"


    static enableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, ENABLED_KEY)
    }

    static disableExternalIps(OrchestratorMain orchestrator) {
        setExternalIps(orchestrator, DISABLED_KEY)
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
