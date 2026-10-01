package util

import com.google.gson.Gson
import groovy.transform.CompileStatic
import io.grpc.CallOptions
import io.grpc.Channel
import io.grpc.ClientCall
import io.grpc.ClientInterceptor
import io.grpc.MethodDescriptor
import org.slf4j.MDC

import java.time.Instant
import java.util.concurrent.ConcurrentHashMap

/** Opt-in, payload-free observations for auditing Groovy spec parallelism. */
@CompileStatic
final class E2EParallelizationAudit {

    private static final String PREFIX = "e2e_parallel_audit "
    private static final String SPEC_KEY = "parallelAuditSpec"
    private static final String EXECUTION_KEY = "parallelAuditExecutionId"
    private static final String TEST_KEY = "parallelAuditTestName"
    private static final Gson JSON = new Gson()
    private static final boolean ENABLED = ["1", "true", "yes", "on"].contains(
            (System.getenv("E2E_PARALLELIZATION_AUDIT_ENABLED") ?: "").toLowerCase(Locale.ROOT))
    private static final Set<String> EMITTED_OPERATIONS = Collections.newSetFromMap(
            new ConcurrentHashMap<String, Boolean>())
    private static final Map<String, String> ACTIVE_SPECIFICATIONS = new ConcurrentHashMap<String, String>()
    private static final ClientInterceptor CENTRAL_INTERCEPTOR = new CentralAuditInterceptor()

    private E2EParallelizationAudit() { }

    static boolean isEnabled() {
        return ENABLED
    }

    static void specStarted(String specification) {
        if (!ENABLED) {
            return
        }
        String executionId = String.valueOf(UUID.randomUUID())
        ACTIVE_SPECIFICATIONS.put(specification, executionId)
        MDC.put(SPEC_KEY, specification)
        MDC.put(EXECUTION_KEY, executionId)
        MDC.remove(TEST_KEY)
        emit("spec_start", [:])
    }

    static void specFinished(String specification) {
        if (ENABLED) {
            String executionId = ACTIVE_SPECIFICATIONS.remove(specification)
            if (executionId != null) {
                MDC.put(SPEC_KEY, specification)
                MDC.put(EXECUTION_KEY, executionId)
                emit("spec_end", [:])
            }
        }
        MDC.remove(TEST_KEY)
        MDC.remove(EXECUTION_KEY)
        MDC.remove(SPEC_KEY)
    }

    static void featureStarted(String specification, String testName) {
        if (ENABLED) {
            MDC.put(SPEC_KEY, specification)
            MDC.remove(EXECUTION_KEY)
            MDC.remove(TEST_KEY)
            String executionId = ACTIVE_SPECIFICATIONS.get(specification)
            if (executionId != null) {
                MDC.put(EXECUTION_KEY, executionId)
            }
            if (testName) {
                MDC.put(TEST_KEY, testName)
            }
        }
    }

    static void featureFinished() {
        MDC.remove(TEST_KEY)
    }

    static void kubernetesRequest(String method, String path) {
        if (!ENABLED) {
            return
        }
        emitOperation("kubernetes", [method: method ?: "unknown", path: path ?: ""])
    }

    static void centralCall(String fullMethodName) {
        if (!ENABLED) {
            return
        }
        int separator = fullMethodName == null ? -1 : fullMethodName.lastIndexOf("/")
        emitOperation("central-grpc", [
                service: separator > 0 ? fullMethodName.substring(0, separator) : "unknown",
                method : separator >= 0 ? fullMethodName.substring(separator + 1) : (fullMethodName ?: "unknown"),
        ])
    }

    /** Records that shared process-wide auth configuration changed; never records its value. */
    static void centralAuthConfigurationChanged() {
        if (ENABLED) {
            emit("operation", [system: "process-global", key: "central-auth-configuration", access: "write"])
        }
    }

    static ClientInterceptor centralClientInterceptor() {
        return CENTRAL_INTERCEPTOR
    }

    private static void emitOperation(String system, Map<String, String> attributes) {
        String key = [
                MDC.get(EXECUTION_KEY) ?: "unattributed",
                MDC.get(TEST_KEY) ?: "",
                system,
                JSON.toJson(attributes),
        ].join("\u0000")
        if (EMITTED_OPERATIONS.add(key)) {
            emit("operation", [system: system] + attributes)
        }
    }

    @SuppressWarnings('SystemOutPrint')
    private static void emit(String eventType, Map<String, String> attributes) {
        try {
            Map<String, Object> record = [
                    schema_version: 1,
                    run_id        : System.getenv("E2E_TIMING_RUN_ID") ?: "local",
                    lane_id       : System.getenv("E2E_TIMING_LANE_ID") ?: "unknown-lane",
                    event_type    : eventType,
                    timestamp     : Instant.now().toString(),
                    execution_id  : MDC.get(EXECUTION_KEY),
                    specification : MDC.get(SPEC_KEY),
                    test_name     : MDC.get(TEST_KEY),
                    attributes    : attributes,
            ]
            synchronized (System.out) {
                System.out.println(PREFIX + JSON.toJson(record))
            }
        } catch (Exception ignored) {
            // The optional audit must not change QA test behavior.
        }
    }

    private static class CentralAuditInterceptor implements ClientInterceptor {
        @Override
        <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
                MethodDescriptor<ReqT, RespT> method,
                CallOptions callOptions,
                Channel next
        ) {
            centralCall(method.fullMethodName)
            return next.newCall(method, callOptions)
        }
    }
}
