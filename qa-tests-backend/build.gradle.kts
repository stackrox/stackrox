import groovy.json.JsonOutput
import org.gradle.api.tasks.testing.TestDescriptor
import org.gradle.api.tasks.testing.TestListener
import org.gradle.api.tasks.testing.TestResult
import org.gradle.api.tasks.testing.logging.TestExceptionFormat
import java.time.Duration
import java.time.Instant
import java.util.UUID
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.ConcurrentLinkedDeque

plugins {
    alias(libs.plugins.protobuf)
    groovy
    codenarc
}

codenarc {
    configFile = file("./codenarc-rules.groovy")
    reportFormat = "text"
}

apply(from = "protobuf.gradle")

// Assign all Java source dirs to Groovy, as the groovy compiler should take care of them.
project.sourceSets.forEach { sourceSet ->
    sourceSet.java.srcDirs.forEach { dir -> sourceSet.groovy.srcDir(dir) }
    sourceSet.java.setSrcDirs(emptyList<File>())
}

dependencies {
    // grpc and protobuf
    implementation(libs.gson)
    implementation(libs.proto.google.common.protos)
    implementation(libs.grpc.alts)
    implementation(libs.grpc.netty)
    implementation(libs.grpc.protobuf)
    implementation(libs.grpc.stub)
    implementation(libs.grpc.auth)
    implementation(libs.netty.tcnative.boringssl.static)

    implementation(platform(libs.groovy.bom))
    implementation(libs.groovy)
    implementation(platform(libs.spock.bom))
    implementation(libs.spock.core)
    implementation(libs.spock.junit4)
    // Gradle 9 requires the JUnit Platform launcher explicitly.
    runtimeOnly("org.junit.platform:junit-platform-launcher")
    implementation(libs.rest.assured)
    testImplementation(libs.snakeyaml)
    implementation(libs.logback.classic)
    testImplementation(libs.jackson.core)
    testImplementation(libs.jackson.annotations)
    testImplementation(libs.jackson.databind)
    implementation(libs.protobuf.java)
    implementation(libs.protobuf.java.util)

    // Use the Kubernetes API
    implementation(libs.kubernetes.client)
    implementation(libs.openshift.client)

    implementation(libs.client.java)

    implementation(libs.commons.httpclient)
    implementation(libs.httpclient)

    implementation(libs.opencsv)

    implementation(libs.commons.cli)

    implementation(libs.commons.exec)

    //JavaMail for mail verifications
    implementation(libs.javax.mail)

    //Slack API
    implementation(libs.slack.api.client)

    // JAX-B dependencies for JDK 9+
    implementation(libs.jaxb.api)
    implementation(libs.jaxb.runtime)

    // Required to make codenarc work with JDK 14.
    // See https://github.com/gradle/gradle/issues/12646.
    constraints {
        codenarc("org.codehaus.groovy:groovy:2.5.10")
        codenarc("org.codehaus.groovy:groovy-xml:2.5.23")
    }

    implementation(libs.javers.core)
    implementation(libs.picocontainer)

    implementation(libs.commons.codec)
    implementation(libs.uuid.creator)

    implementation(projects.annotations)
}

private val activeTestTimings = ConcurrentHashMap<String, ConcurrentLinkedDeque<String>>()

private fun e2eTimingEnabled(): Boolean =
    System.getenv("E2E_TIMING_ENABLED")?.lowercase() in setOf("1", "true", "yes", "on")

private fun emitTestTimingEvent(
    phase: String,
    name: String,
    spanId: String,
    event: String,
    timestamp: String,
    attributes: Map<String, String>,
    outcome: String? = null,
    reason: String? = null,
) {
    if (!e2eTimingEnabled()) {
        return
    }

    val record = linkedMapOf<String, Any>(
        "schema_version" to 1,
        "run_id" to (System.getenv("E2E_TIMING_RUN_ID") ?: "local"),
        "lane_id" to (System.getenv("E2E_TIMING_LANE_ID") ?: "unknown-lane"),
        "provider" to (System.getenv("E2E_TIMING_PROVIDER") ?: "local"),
        "span_id" to spanId,
        "phase" to phase,
        "name" to name,
        "event" to event,
        "timestamp" to timestamp,
        "attributes" to attributes,
    )
    if (outcome != null) {
        record["outcome"] = outcome
    }
    if (reason != null) {
        record["reason"] = reason
    }

    try {
        synchronized(System.out) {
            System.out.println("e2e_timing ${JsonOutput.toJson(record)}")
        }
    } catch (_: Exception) {
        // Timing output must not affect test execution.
    }
}

private fun testTimingKey(phase: String, taskPath: String, descriptor: TestDescriptor): String =
    "$phase\u0000$taskPath\u0000${descriptor.className ?: "unknown-class"}\u0000${descriptor.name}"

private fun testTimingName(phase: String, taskName: String, descriptor: TestDescriptor): String {
    val name = "groovy:$taskName:${descriptor.className ?: "unknown-class"}"
    return if (phase == "test-case") "$name::${descriptor.name}" else name
}

private fun testTimingAttributes(
    phase: String,
    taskName: String,
    taskPath: String,
    descriptor: TestDescriptor,
): Map<String, String> = mapOf(
    "framework" to "spock",
    "gradle_task" to taskName,
    "gradle_task_path" to taskPath,
    "test_class" to (descriptor.className ?: "unknown-class"),
) + if (phase == "test-case") mapOf("test_name" to descriptor.name) else emptyMap()

private fun startTestTiming(
    phase: String,
    taskName: String,
    taskPath: String,
    descriptor: TestDescriptor,
) {
    if (!e2eTimingEnabled()) {
        return
    }
    val spanId = "$phase:${UUID.randomUUID()}"
    activeTestTimings.computeIfAbsent(testTimingKey(phase, taskPath, descriptor)) {
        ConcurrentLinkedDeque()
    }.addLast(spanId)
    emitTestTimingEvent(
        phase = phase,
        name = testTimingName(phase, taskName, descriptor),
        spanId = spanId,
        event = "start",
        timestamp = Instant.now().toString(),
        attributes = testTimingAttributes(phase, taskName, taskPath, descriptor),
    )
}

private fun finishTestTiming(
    phase: String,
    taskName: String,
    taskPath: String,
    descriptor: TestDescriptor,
    result: TestResult,
) {
    if (!e2eTimingEnabled()) {
        return
    }
    val key = testTimingKey(phase, taskPath, descriptor)
    val timings = activeTestTimings[key]
    val spanId = timings?.pollLast()
    if (timings != null && timings.isEmpty()) {
        activeTestTimings.remove(key, timings)
    }

    val attributes = testTimingAttributes(phase, taskName, taskPath, descriptor)
    val name = testTimingName(phase, taskName, descriptor)
    if (spanId == null) {
        // A filtered/skipped test has no start event; don't fabricate a zero-length span.
        emitTestTimingEvent(
            phase = phase,
            name = name,
            spanId = "$phase:${UUID.randomUUID()}",
            event = "skipped",
            timestamp = Instant.now().toString(),
            attributes = attributes,
            reason = "gradle-result-without-start-event",
        )
        return
    }

    val outcome = when (result.resultType.name) {
        "SUCCESS" -> "success"
        "FAILURE" -> "failure"
        else -> "skipped"
    }
    emitTestTimingEvent(
        phase = phase,
        name = name,
        spanId = spanId,
        event = "end",
        timestamp = Instant.now().toString(),
        attributes = attributes,
        outcome = outcome,
    )
}

tasks.withType<GroovyCompile>().configureEach {
    groovyOptions.forkOptions.memoryMaximumSize = "4g"
}

// Apply some base attributes to all the test tasks.
tasks.withType<Test>().configureEach {
    val timingTaskName = name
    val timingTaskPath = path

    testLogging {
        showStandardStreams = true
        exceptionFormat = TestExceptionFormat.FULL
        events("passed", "skipped", "failed")
    }
    timeout = Duration.ofMinutes(630)

    // This ensures that repeated invocations of tests actually run the tests.
    // Otherwise, if the tests pass, Gradle "caches" the result and doesn"t actually run the tests,
    // which is not the behaviour we expect of E2Es.
    // https://stackoverflow.com/questions/42175235/force-gradle-to-run-task-even-if-it-is-up-to-date/42185919
    outputs.upToDateWhen { false }

    reports {
        junitXml.isOutputPerTestCase = true
        junitXml.mergeReruns = true
    }

    useJUnitPlatform();

    // Gradle 9: registered Test tasks (testBAT, testSMOKE, etc.) don't
    // inherit testClassesDirs from the test source set. Wire explicitly
    // so test discovery finds compiled classes.
    val testSourceSet = project.sourceSets["test"]
    testClassesDirs = testSourceSet.output.classesDirs
    classpath = testSourceSet.runtimeClasspath

    // Catches the case when tag filters match nothing: the task runs but
    // zero tests execute. This is different case than NO-SOURCE handled below with afterTask hook.
    addTestListener(object : TestListener {
        override fun beforeTest(test: TestDescriptor) {
            startTestTiming("test-case", timingTaskName, timingTaskPath, test)
        }

        override fun afterTest(test: TestDescriptor, result: TestResult) {
            finishTestTiming("test-case", timingTaskName, timingTaskPath, test, result)
        }

        override fun beforeSuite(suite: TestDescriptor) {
            // Gradle also reports task and worker suites; only class suites add
            // fixture/setup time that is outside the individual test-case spans.
            if (suite.isComposite && suite.className != null) {
                startTestTiming("test-suite", timingTaskName, timingTaskPath, suite)
            }
        }

        override fun afterSuite(suite: TestDescriptor, result: TestResult) {
            if (suite.isComposite && suite.className != null) {
                finishTestTiming("test-suite", timingTaskName, timingTaskPath, suite, result)
            }
            if (suite.parent == null && result.testCount == 0L) {
                throw GradleException(
                    "No tests were executed in task '${name}'. " +
                    "This likely means no tests matched the configured filters. " +
                    "Check your tests selection (e.g. includeTags/excludeTags)."
                )
            }
        }
    })
}

tasks.register<Test>("testParallel") {
    systemProperty("spock.configuration", rootProject.file("src/test/resources/ParallelSpockConfig.groovy"))
    useJUnitPlatform {
        includeTags("Parallel")
    }
}

tasks.register<Test>("testRest") {
    useJUnitPlatform {
        excludeTags("Parallel", "Upgrade", "SensorBounce", "SensorBounceNext")
    }
}

tasks.register<Test>("testParallelBAT") {
    systemProperty("spock.configuration", rootProject.file("src/test/resources/ParallelSpockConfig.groovy"))
    useJUnitPlatform {
        includeTags("Parallel & BAT")
    }
}

tasks.register<Test>("testBAT") {
    useJUnitPlatform {
        includeTags("BAT")
        excludeTags("Parallel")
    }
}

tasks.register<Test>("testSMOKE") {
    useJUnitPlatform {
        includeTags("SMOKE")
    }
}

tasks.register<Test>("testCOMPATIBILITY") {
    useJUnitPlatform {
        includeTags("COMPATIBILITY")
        excludeTags("SensorBounce")
    }
}

tasks.register<Test>("testCOMPATIBILITYSensorBounce") {
    useJUnitPlatform {
        includeTags("COMPATIBILITY & SensorBounce")
    }
}

tasks.register<Test>("testRUNTIME") {
    useJUnitPlatform {
        includeTags("RUNTIME")
    }
}

tasks.register<Test>("testPolicyEnforcement") {
    useJUnitPlatform {
        includeTags("PolicyEnforcement")
    }
}

tasks.register<Test>("testIntegration") {
    useJUnitPlatform {
        includeTags("Integration")
    }
}

tasks.register<Test>("testNetworkPolicySimulation") {
    useJUnitPlatform {
        includeTags("NetworkPolicySimulation")
    }
}

tasks.register<Test>("testUpgrade") {
    useJUnitPlatform {
        includeTags("Upgrade")
    }
}

tasks.register<Test>("testGraphQL") {
    useJUnitPlatform {
        includeTags("GraphQL")
    }
}

tasks.register<Test>("testSensorBounce") {
    useJUnitPlatform {
        includeTags("SensorBounce")
    }
}

tasks.register<Test>("testSensorBounceNext") {
    useJUnitPlatform {
        includeTags("SensorBounceNext")
    }
}

tasks.register<JavaExec>("runSampleScript") {
    dependsOn("classes")
    if (project.hasProperty("runScript")) {
        mainClass = "sampleScripts." + project.properties["runScript"]
        classpath = sourceSets["main"].runtimeClasspath
    }
}

tasks.register<Test>("testPZ") {
    useJUnitPlatform {
        includeTags("PZ")
    }
}

tasks.register<Test>("testPZDebug") {
    useJUnitPlatform {
        includeTags("PZDebug")
    }
}

tasks.register<Test>("testDeploymentCheck") {
    useJUnitPlatform {
        includeTags("DeploymentCheck")
    }
}

// Fail the build if any Test task is skipped with NO-SOURCE. Gradle exits 0
// in this case, silently producing no test results. afterSuite/doFirst/doLast
// callbacks don't fire for NO-SOURCE tasks, so this project-level hook is
// the only reliable way to detect it.
// https://discuss.gradle.org/t/copy-task-how-to-fail-on-no-source/25581
// https://github.com/gradle/gradle/issues/36700
//
// afterTask is deprecated since Gradle 8.3 and incompatible with configuration
// cache (explicitly disabled in gradle.properties). No replacement exists for
// detecting NO-SOURCE skips — tracked in gradle/gradle#36700.
@Suppress("DEPRECATION")
gradle.taskGraph.afterTask {
    if (this is Test && this.state.noSource) {
        throw GradleException(
            "Test task '${this.path}' was skipped with NO-SOURCE. " +
            "Check that testClassesDirs and classpath are wired correctly."
        )
    }
}

allprojects {
    apply(plugin = "java")
    java {
        toolchain {
            languageVersion = JavaLanguageVersion.of(17)
        }
    }
}
