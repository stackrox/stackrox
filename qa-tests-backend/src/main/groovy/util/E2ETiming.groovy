package util

import com.google.gson.Gson

import java.time.Instant

/** Best-effort activity spans for correlating shared QA helpers with test cases. */
final class E2ETiming {

    private static final Gson JSON = new Gson()
    private static final Set<String> TRUE_VALUES = ["1", "true", "yes", "on"] as Set<String>

    private E2ETiming() { }

    // Rethrow assertions/errors after recording them so the activity outcome matches the test result.
    @SuppressWarnings('CatchThrowable')
    static <T> T measure(String category, String helper, Map<String, String> details, Closure<T> action) {
        if (!enabled()) {
            return action.call()
        }

        String spanId = "test-activity:${UUID.randomUUID()}"
        String name = "activity:${category}:${helper}"
        Map<String, String> attributes = [
                framework: "spock",
                activity : category,
                helper   : helper,
        ]
        attributes.putAll(details)
        emit(spanId, name, "start", attributes, null)

        String outcome = "success"
        try {
            return action.call()
        } catch (Throwable failure) {
            outcome = "failure"
            throw failure
        } finally {
            // Callers can add observed values (for example, retry attempts) while the span is open.
            attributes.putAll(details)
            emit(spanId, name, "end", attributes, outcome)
        }
    }

    private static boolean enabled() {
        return TRUE_VALUES.contains((System.getenv("E2E_TIMING_ENABLED") ?: "").toLowerCase(Locale.ROOT))
    }

    // The timing extractor consumes this unadorned JSON sentinel from test stdout.
    @SuppressWarnings('SystemOutPrint')
    private static void emit(
            String spanId,
            String name,
            String event,
            Map<String, String> attributes,
            String outcome
    ) {
        try {
            Map<String, Object> record = [
                    schema_version: 1,
                    run_id        : System.getenv("E2E_TIMING_RUN_ID") ?: "local",
                    lane_id       : System.getenv("E2E_TIMING_LANE_ID") ?: "unknown-lane",
                    provider      : System.getenv("E2E_TIMING_PROVIDER") ?: "local",
                    span_id       : spanId,
                    phase         : "test-activity",
                    name          : name,
                    event         : event,
                    timestamp     : Instant.now().toString(),
                    attributes    : attributes,
            ]
            if (outcome != null) {
                record.outcome = outcome
            }
            synchronized (System.out) {
                System.out.println("e2e_timing ${JSON.toJson(record)}")
            }
        } catch (Exception ignored) {
            // Timing output must not change QA test behavior.
        }
    }
}
