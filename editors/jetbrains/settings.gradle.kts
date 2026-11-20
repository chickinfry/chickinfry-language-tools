// The IntelliJ Platform Gradle plugin pulls okhttp, retrofit, and xmlutil into
// the buildscript classpath. Resolving those through the plugin portal alone
// has failed intermittently with "could not find", while the same artifacts
// answer 200 from Maven Central. Naming Maven Central here lets the classpath
// resolve directly.
pluginManagement {
    repositories {
        gradlePluginPortal()
        mavenCentral()
    }
}

rootProject.name = "chik"
