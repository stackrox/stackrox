package objects

import com.google.gson.annotations.SerializedName

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
        public SplunkFeatureHealth features
        public String health
        static class SplunkFeatureHealth {
            @SerializedName("Index Processor") public SplunkIndexProcessorHealth indexProcessor
            @SerializedName("Search Scheduler") public SplunkSearchSchedulerHealth searchScheduler
            static class SplunkIndexProcessorHealth {
                public String health
            }
            static class SplunkSearchSchedulerHealth {
                public String health
            }
        }
    }
}
