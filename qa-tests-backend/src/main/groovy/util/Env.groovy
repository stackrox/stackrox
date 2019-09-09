package util

import orchestratormanager.OrchestratorTypes

class Env {

    private static String mustGetEnv(String envVar) {
        String value = System.getenv(envVar)
        if (!value) {
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

    static String mustGetKeystorePath() {
        return mustGetEnv("KEYSTORE_PATH")
    }

    static String mustGetTruststorePath() {
        return mustGetEnv("TRUSTSTORE_PATH")
    }

    static String mustGetClientCAPath() {
        return mustGetEnv("CLIENT_CA_PATH")
    }

    static String mustGetCiJobName() {
        return mustGetEnv("CI_JOB_NAME")
    }

    static String mustGetImageTag() {
        return mustGetEnv("IMAGE_TAG")
    }

    static String mustGetResultsFilePath() {
        return mustGetEnv("RESULTS_FILE_PATH")
    }

    static String mustGetTestRailPassword() {
        return mustGetEnv("TESTRAIL_PASSWORD")
    }
}

