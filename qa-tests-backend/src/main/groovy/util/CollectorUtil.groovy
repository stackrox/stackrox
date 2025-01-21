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
    String runtime_config = """\
networking:
  externalIps:
    enabled: ${state}
"""
        def Map<String, String> map_data = [
            (RUNTIME_CONFIG_MAP_KEY): runtime_config,
        ]

        orchestrator.createConfigMap(RUNTIME_CONFIG_MAP_NAME, map_data, "stackrox")
    }
}
