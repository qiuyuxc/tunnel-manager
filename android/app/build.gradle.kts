plugins {
    id("com.android.application")
}

android {
    namespace = "com.tunnelmanager.app"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.tunnelmanager.app"
        minSdk = 26
        targetSdk = 34
        versionCode = 241
        versionName = "2.4.1"
    }

    // The debug key is the one build.sh already generated, so `adb install -r`
    // keeps working across both build paths.
    signingConfigs {
        create("shared") {
            storeFile = file(System.getenv("ANDROID_DEBUG_KEYSTORE")
                ?: "${System.getProperty("user.home")}/.android/debug.keystore")
            storePassword = "android"
            keyAlias = "androiddebugkey"
            keyPassword = "android"
        }
    }

    buildTypes {
        getByName("debug") {
            signingConfig = signingConfigs.getByName("shared")
        }
        getByName("release") {
            isMinifyEnabled = false
            signingConfig = signingConfigs.getByName("shared")
        }
    }

    buildFeatures {
        viewBinding = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("com.google.android.material:material:1.12.0")
    implementation("androidx.constraintlayout:constraintlayout:2.2.0")
    implementation("androidx.recyclerview:recyclerview:1.3.2")
    implementation("androidx.swiperefreshlayout:swiperefreshlayout:1.1.0")
}
