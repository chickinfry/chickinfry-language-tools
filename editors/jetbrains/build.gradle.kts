import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.4.21"
    id("org.jetbrains.intellij.platform") version "2.20.1"
}

repositories {
    mavenCentral()
    intellijPlatform {
        defaultRepositories()
    }
}

dependencies {
    intellijPlatform {
        // The LSP API is only present in the commercial IDE distributions.
        intellijIdea(providers.gradleProperty("platformVersion"))
        bundledPlugin("org.jetbrains.plugins.textmate")
        testFramework(TestFrameworkType.Platform)
    }
    testImplementation("junit:junit:4.13.2")
}

intellijPlatform {
    pluginConfiguration {
        id = providers.gradleProperty("pluginGroup")
        name = providers.gradleProperty("pluginName")
        version = providers.gradleProperty("pluginVersion")

        // The Marketplace listing shows this text. It is read from the
        // changelog so the published notes and the file cannot disagree.
        changeNotes = providers.fileContents(
            layout.projectDirectory.file("CHANGELOG.md"),
        ).asText.map { it.lines().drop(2).joinToString("\n").trim() }

        ideaVersion {
            sinceBuild = providers.gradleProperty("pluginSinceBuild")
        }
    }

    // Signing and publishing are done by the release owner, not by CI.
    buildSearchableOptions = false
    instrumentCode = false
}

tasks.withType<Test>().configureEach {
    useJUnit()
}

tasks.processResources {
    from(rootProject.file("../../LICENSE")) {
        into("META-INF")
    }
}

