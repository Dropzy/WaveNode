# Android App Build Instructions

## Prerequisites

Before building the Android app, you need to install the following:

### 1. Java Development Kit (JDK)
- **Required**: JDK 17 or later
- **Download**: [Oracle JDK](https://www.oracle.com/java/technologies/downloads/) or [OpenJDK](https://adoptium.net/)
- **Installation**: 
  - Download and install JDK 17+
  - Set JAVA_HOME environment variable
  - Add Java bin directory to PATH

### 2. Android SDK
- **Required**: Android SDK (API level 24+)
- **Download**: [Android Studio](https://developer.android.com/studio) (includes SDK)
- **Installation**:
  - Install Android Studio
  - Open SDK Manager (Tools > SDK Manager)
  - Install Android SDK Platform 24+ (recommend API 34)
  - Install Android SDK Build-Tools
  - Install Android SDK Platform-Tools

### 3. Environment Variables
Set the following environment variables:
```bash
JAVA_HOME=C:\Program Files\Java\jdk-17
ANDROID_HOME=C:\Users\%USERNAME%\AppData\Local\Android\Sdk
PATH=%PATH%;%JAVA_HOME%\bin;%ANDROID_HOME%\platform-tools;%ANDROID_HOME%\tools
```

## Build Process

### Option 1: Using Android Studio (Recommended)
1. **Open Project**:
   - Launch Android Studio
   - Select "Open an existing project"
   - Navigate to the `Android` folder and open it

2. **Sync Project**:
   - Android Studio will automatically sync the project
   - Wait for Gradle sync to complete (may take several minutes)

3. **Build APK**:
   - Go to **Build > Build Bundle(s) / APK(s) > Build APK(s)**
   - Wait for the build to complete
   - The APK will be located at: `Android/app/build/outputs/apk/debug/app-debug.apk`

### Option 2: Using Command Line
1. **Navigate to Android Directory**:
   ```bash
   cd Android
   ```

2. **Build Debug APK**:
   ```bash
   ./gradlew assembleDebug
   ```
   - On Windows: `gradlew.bat assembleDebug`
   - On Linux/Mac: `./gradlew assembleDebug`

3. **Build Release APK** (for production):
   ```bash
   ./gradlew assembleRelease
   ```

## APK Location
After successful build, the APK files will be located at:
- **Debug**: `Android/app/build/outputs/apk/debug/app-debug.apk`
- **Release**: `Android/app/build/outputs/apk/release/app-release.apk`

## Configuration

### Backend URL
Before building, update the backend URL in:
`Android/app/src/main/java/com/musicserver/data/api/ApiClient.kt`

```kotlin
private const val BASE_URL = "http://your-backend-url:8080/api/"
```

### App Configuration
Update app details in:
`Android/app/build.gradle.kts`
- `applicationId`: Change to your package name
- `versionCode` & `versionName`: Update as needed

## Installation

### On Emulator
1. Start Android Emulator from Android Studio
2. Install APK:
   ```bash
   adb install app-debug.apk
   ```

### On Physical Device
1. Enable Developer Options and USB Debugging
2. Connect device via USB
3. Install APK:
   ```bash
   adb install app-debug.apk
   ```

## Troubleshooting

### Common Issues

1. **JAVA_HOME not set**:
   - Ensure JDK is installed
   - Set JAVA_HOME environment variable correctly

2. **Android SDK not found**:
   - Install Android Studio
   - Set ANDROID_HOME environment variable
   - Install required SDK platforms

3. **Gradle sync fails**:
   - Check internet connection
   - Try File > Invalidate Caches / Restart in Android Studio
   - Ensure all dependencies are available

4. **Build fails with compilation errors**:
   - Check if all required SDK versions are installed
   - Update Gradle and Android Gradle Plugin if needed
   - Clean and rebuild the project

### Useful Gradle Commands
```bash
# Clean project
./gradlew clean

# Build debug APK
./gradlew assembleDebug

# Build release APK (requires signing configuration)
./gradlew assembleRelease

# Install debug APK to connected device/emulator
./gradlew installDebug

# Run tests
./gradlew test

# Generate signed APK (requires keystore configuration)
./gradlew assembleRelease
```

## Release Build (Production)

For production builds, you need to:

1. **Generate Keystore**:
   ```bash
   keytool -genkey -v -keystore release-key.keystore -alias release -keyalg RSA -keysize 2048 -validity 10000
   ```

2. **Configure Signing**:
   Add signing configuration to `app/build.gradle.kts`

3. **Build Signed APK**:
   ```bash
   ./gradlew assembleRelease
   ```

## Project Structure Summary

The Android app includes:
- **Complete authentication system** (login/register)
- **Music player** with ExoPlayer integration
- **All UI screens** matching web frontend functionality
- **Material Design 3** theming
- **Navigation** between all screens
- **API integration** with your existing backend
- **Admin dashboard** for library management

The app is ready to build once you have the required development environment set up.
