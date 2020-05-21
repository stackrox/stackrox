package objects

class SplunkSearch {
    String sid
}

class SplunkAlerts {
    List<SplunkAlertRaw> results
}

class SplunkAlertRaw {
    String _raw
}

class SplunkAlert {
    SplunkAlertPolicy policy
    SplunkAlertDeployment deployment
    List<SplunkAlertViolations> violations
}

class SplunkAlertPolicy {
    String id
    String name
    String description
    String rationale
    String remediation
}

class SplunkAlertDeployment {
    String id
    String name
    String namespace
    String type
}

class SplunkAlertViolations {
    String message
}
