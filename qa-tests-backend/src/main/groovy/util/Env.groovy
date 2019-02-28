package util

import orchestratormanager.OrchestratorTypes

class Env {

    private static String mustGetEnv(String envVar) {
        String value = System.getenv(envVar)
        if (value == "null") {
            throw new RuntimeException(envVar + " must be defined in the env")
        }
        return value
    }

    static int mustGetPort() {
        String portString = mustGetEnv("PORT")
        int port
        try {
            port = Integer.parseInt(portString)
        } catch (NumberFormatException e) {
            throw new RuntimeException("PORT " + portString + " is not a valid number " + e.toString())
        }
        return port
    }

    static String mustGetHostname() {
        return mustGetEnv("HOSTNAME")
    }

    static OrchestratorTypes mustGetOrchestratorType() {
        String cluster = mustGetEnv("CLUSTER")
        OrchestratorTypes type
        try {
            type = OrchestratorTypes.valueOf(cluster)
        } catch (IllegalArgumentException e) {
            throw new RuntimeException("CLUSTER must be one of " + OrchestratorTypes.values() + " " + e.toString())
        }
        return type
    }
}

