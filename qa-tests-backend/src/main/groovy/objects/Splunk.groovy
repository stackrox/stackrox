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

class SplunkHECTokens {
    List<SplunkHECEntry> entry
}

// codenarc-disable PublicInstanceField
class SplunkHECEntry {
    public SplunkHECContent content
    static class SplunkHECContent {
        public String token
    }
}

class SplunkHealthResults {
    List<SplunkHealthEntry> entry
}

// codenarc-disable PublicInstanceField
class SplunkHealthEntry {
    public SplunkHealthContent content
    static class SplunkHealthContent {
        public String health
    }
}