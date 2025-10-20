import org.gradle.api.tasks.testing.logging.TestExceptionFormat
import java.time.Duration

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
    sourceSet.groovy.srcDirs += sourceSet.java.srcDirs
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

    implementation(projects.annotations)
}

// Apply some base attributes to all the test tasks.
tasks.withType<Test>().configureEach {
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

    useJUnitPlatform()
}

data class TestTask(
    var includeTags: Set<String> = setOf(),
    var excludeTags: Set<String> = setOf(),
    var spockConfiguration: File? = null,
)

val tests = mapOf(
    "testBegin" to TestTask(includeTags = setOf("Begin")),
    "testRest" to TestTask(includeTags = setOf("Begin", "Parallel", "Upgrade", "SensorBounce", "SensorBounceNext")),
    "testBAT" to TestTask(includeTags = setOf("BAT"), excludeTags = setOf("Parallel")),
    "testSMOKE" to TestTask(includeTags = setOf("SMOKE")),
    "testCOMPATIBILITY" to TestTask(includeTags = setOf("COMPATIBILITY"), excludeTags = setOf("SensorBounce")),
    "testCOMPATIBILITYSensorBounce" to TestTask(includeTags = setOf("COMPATIBILITY & SensorBounce")),
    "testRUNTIME" to TestTask(includeTags = setOf("RUNTIME")),
    "testPolicyEnforcement" to TestTask(includeTags = setOf("PolicyEnforcement")),
    "testIntegration" to TestTask(includeTags = setOf("Integration")),
    "testNetworkPolicySimulation" to TestTask(includeTags = setOf("NetworkPolicySimulation")),
    "testUpgrade" to TestTask(includeTags = setOf("Upgrade")),
    "testGraphQL" to TestTask(includeTags = setOf("GraphQL")),
    "testSensorBounce" to TestTask(includeTags = setOf("SensorBounce")),
    "testSensorBounceNext" to TestTask(includeTags = setOf("SensorBounceNext")),
    "testPZ" to TestTask(includeTags = setOf("PZ")),
    "testPZDebug" to TestTask(includeTags = setOf("PZDebug")),
    "testDeploymentCheck" to TestTask(includeTags = setOf("DeploymentCheck")),
    "testParallel" to TestTask(
        includeTags = setOf("Parallel"),
        spockConfiguration = rootProject.file("src/test/resources/ParallelSpockConfig.groovy")
    ),
    "testParallelBAT" to TestTask(
        includeTags = setOf("Parallel & BAT"),
        spockConfiguration = rootProject.file("src/test/resources/ParallelSpockConfig.groovy")
    ),
).forEach { (name, testTask) ->
    tasks.register<Test>(name) {
        useJUnitPlatform {
            includeTags(*testTask.includeTags.toTypedArray<String>())
            excludeTags(*testTask.excludeTags.toTypedArray<String>())
            testTask.spockConfiguration?.let {
                systemProperty(
                    "spock.configuration",
                    testTask.spockConfiguration as Any
                )
            }
        }
    }
}

tasks.register<JavaExec>("runSampleScript") {
    dependsOn("classes")
    if (project.hasProperty("runScript")) {
        mainClass = "sampleScripts." + project.properties["runScript"]
        classpath = sourceSets["main"].runtimeClasspath
    }
}

allprojects {
    apply(plugin = "java")
    java {
        toolchain {
            languageVersion.set(JavaLanguageVersion.of(17))
        }
    }
}
