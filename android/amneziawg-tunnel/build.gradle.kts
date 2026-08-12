@file:Suppress("UnstableApiUsage")

plugins {
    id("com.android.library")
}

val upstream = layout.projectDirectory.dir("../vendor/amneziawg-android/tunnel")
val generatedLicenses = layout.buildDirectory.dir("generated/third-party-assets/licenses")
val prepareThirdPartyLicenses by tasks.registering(Copy::class) {
    from(layout.projectDirectory.file("../vendor/amneziawg-android/COPYING")) {
        rename { "amneziawg-android-APACHE-2.0.txt" }
    }
    from(layout.projectDirectory.file("native/libwg-go/LICENSE")) {
        rename { "amneziawg-go-MIT.txt" }
    }
    into(generatedLicenses)
}

android {
    namespace = "org.amnezia.awg.tunnel"
    compileSdk = 36
    ndkVersion = "26.1.10909125"

    defaultConfig {
        minSdk = 26
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    sourceSets.named("main") {
        manifest.srcFile("src/main/AndroidManifest.xml")
        java.srcDir(upstream.dir("src/main/java"))
        java.exclude("org/amnezia/awg/backend/**")
        java.exclude("org/amnezia/awg/util/RootShell.java")
        java.exclude("org/amnezia/awg/util/SharedLibraryLoader.java")
        java.exclude("org/amnezia/awg/util/ToolsInstaller.java")
        assets.srcDir(generatedLicenses)
    }
    externalNativeBuild {
        cmake {
            path = file("native/CMakeLists.txt")
        }
    }
    buildTypes {
        all {
            externalNativeBuild {
                cmake {
                    targets("libwg-go.so")
                    arguments(
                        "-DGRADLE_USER_HOME=${project.gradle.gradleUserHomeDir}",
                        "-DANDROID_PACKAGE_NAME=com.mazzy.vpn"
                    )
                }
            }
        }
    }
}

tasks.named("preBuild").configure { dependsOn(prepareThirdPartyLicenses) }

dependencies {
    implementation("androidx.annotation:annotation:1.9.1")
    implementation("androidx.collection:collection:1.5.0")
    compileOnly("com.google.code.findbugs:jsr305:3.0.2")
    testImplementation("junit:junit:4.13.2")
}
