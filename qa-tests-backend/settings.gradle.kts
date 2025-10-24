dependencyResolutionManagement {
    repositories {
        mavenLocal()
        mavenCentral()
    }
}

enableFeaturePreview("TYPESAFE_PROJECT_ACCESSORS")

rootProject.name = "qa-tests-backend"
include("annotations")
