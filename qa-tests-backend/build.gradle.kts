import org.gradle.api.tasks.testing.logging.TestExceptionFormat
import java.time.Duration

plugins {
    alias(libs.plugins.protobuf)
    groovy
    codenarc
}

version = "1.0"

codenarc.configFile = file("./codenarc-rules.groovy")
codenarc.reportFormat = "text"

apply(from = "protobuf.gradle")

// Assign all Java source dirs to Groovy, as the groovy compiler should take care of them.
project.sourceSets.forEach { sourceSet ->
    sourceSet.groovy.srcDirs += sourceSet.java.srcDirs
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
        "codenarc"("org.codehaus.groovy:groovy:2.5.10")
        "codenarc"("org.codehaus.groovy:groovy-xml:2.5.23")
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

    useJUnitPlatform {
        includeTags("Parallel & BAT")
    }
}

tasks.register<Test>("testBegin") {
    useJUnitPlatform {
        includeTags("Begin")
    }
}

tasks.register<Test>("testParallel") {
    systemProperty("spock.configuration", rootProject.file("src/test/resources/ParallelSpockConfig.groovy"))
    useJUnitPlatform {
        includeTags("Parallel")
    }
}

tasks.register<Test>("testRest") {
    useJUnitPlatform {
        excludeTags("Begin", "Parallel", "Upgrade", "SensorBounce", "SensorBounceNext")
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

allprojects {
    apply(plugin = "java")
    group = "io.stackrox"
    java {
        toolchain {
            languageVersion.set(JavaLanguageVersion.of(11))
        }
    }
}
