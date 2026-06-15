# Music Server Android App

A native Android application that provides the same functionality as the web frontend for the Music Server backend. Built with modern Android development practices using Jetpack Compose, Hilt for dependency injection, and ExoPlayer for audio playback.

## Features

- **Authentication**: Login and register functionality with JWT token management
- **Music Library**: Browse and search through your music collection
- **Music Player**: Full-featured audio player with ExoPlayer integration
- **Playlists**: Create and manage custom playlists
- **Album & Artist Views**: Detailed views for albums and artists
- **Liked Songs**: Save and manage your favorite tracks
- **Admin Dashboard**: Administrative interface for managing the music library
- **Offline Support**: Cache music for offline playback (planned)
- **Material Design 3**: Modern UI following Material Design 3 guidelines

## Architecture

The app follows clean architecture principles with the following layers:

### Data Layer
- **Models**: Data classes for API responses and local storage
- **API Services**: Retrofit interfaces for backend communication
- **Repositories**: Data repositories that abstract data sources
- **Local Storage**: SharedPreferences for token management

### Domain Layer
- **Use Cases**: Business logic and data transformation (simplified in ViewModels)

### Presentation Layer
- **UI Screens**: Jetpack Compose screens for each feature
- **ViewModels**: State management and business logic
- **Navigation**: Navigation Component for screen transitions

## Technology Stack

- **UI**: Jetpack Compose
- **Architecture**: MVVM with Hilt dependency injection
- **Networking**: Retrofit2 with OkHttp3
- **Audio**: ExoPlayer for music playback
- **Navigation**: Navigation Component for Compose
- **Async**: Coroutines and Flow
- **Image Loading**: Coil (for future album art support)
- **Build System**: Gradle with Kotlin DSL

## Project Structure

```
Android/
├── app/
│   ├── build.gradle.kts          # App-level build configuration
│   ├── src/main/
│   │   ├── AndroidManifest.xml   # App manifest
│   │   ├── java/com/musicserver/
│   │   │   ├── data/             # Data layer
│   │   │   │   ├── api/          # API services and client
│   │   │   │   ├── local/        # Local storage
│   │   │   │   ├── models/       # Data models
│   │   │   │   └── repository/   # Repository implementations
│   │   │   ├── di/               # Dependency injection modules
│   │   │   ├── navigation/       # Navigation setup
│   │   │   ├── player/           # Music service and player
│   │   │   ├── ui/               # UI layer
│   │   │   │   ├── screens/      # Compose screens
│   │   │   │   └── theme/        # Material Design 3 theme
│   │   │   ├── viewmodel/        # ViewModels
│   │   │   ├── MainActivity.kt   # Main activity
│   │   │   └── MusicApplication.kt # Application class
│   │   └── res/                  # Android resources
│   └── build.gradle.kts          # Project-level build configuration
├── build.gradle.kts              # Root build configuration
├── gradle.properties             # Gradle properties
└── settings.gradle.kts           # Gradle settings
```

## Setup Instructions

### Prerequisites

- Android Studio Hedgehog | 2023.1.1 or later
- Android SDK (API 24+)
- Java 17 or later
- Kotlin 1.9.10 or later

### Configuration

1. **Backend URL Configuration**:
   Update the base URL in `ApiClient.kt`:
   ```kotlin
   private const val BASE_URL = "http://your-backend-url:8080/api/"
   ```

2. **Build the App**:
   ```bash
   ./gradlew assembleDebug
   ```

3. **Run on Emulator/Device**:
   ```bash
   ./gradlew installDebug
   ```

### Dependencies

Key dependencies are managed in `app/build.gradle.kts`:

- **Compose UI**: `androidx.compose.*`
- **Navigation**: `androidx.navigation.compose`
- **Hilt**: `com.google.dagger:hilt-*`
- **Retrofit**: `com.squareup.retrofit2:*`
- **ExoPlayer**: `androidx.media3:media3-exoplayer`
- **Coroutines**: `org.jetbrains.kotlinx:kotlinx-coroutines-*`

## Key Components

### MusicService
A foreground service that handles music playback using ExoPlayer. Features include:
- Background playback
- Media session integration
- Notification controls
- Playlist management

### Authentication Flow
- JWT token storage in SharedPreferences
- Automatic token refresh
- Protected routes with authentication interceptors

### Navigation
Single-activity architecture with Navigation Component:
- Login/Register flow
- Main app navigation
- Deep linking support for albums, artists, and playlists

### State Management
- ViewModels with StateFlow for reactive UI
- Hilt for dependency injection
- Coroutines for async operations

## API Integration

The app communicates with the same backend as the web frontend:

- **Authentication**: `/api/auth/login`, `/api/auth/register`
- **Music Library**: `/api/music/*`
- **Playlists**: `/api/playlists/*`
- **User Management**: `/api/users/*`

## Future Enhancements

- [ ] Offline music caching
- [ ] Album art display with Coil
- [ ] WebSocket integration for real-time updates
- [ ] Chromecast support
- [ ] Equalizer and audio effects
- [ ] Sleep timer
- [ ] Lyrics display
- [ ] Social features (sharing, following)

## Troubleshooting

### Common Issues

1. **Network Connection**: Ensure the backend URL is correctly configured
2. **Audio Playback**: Check if the music files are accessible via the configured URLs
3. **Authentication**: Verify JWT token handling and refresh logic
4. **Build Issues**: Clean and rebuild the project if dependency issues occur

### Debugging

Enable debug logging in `ApiClient.kt`:
```kotlin
val logging = HttpLoggingInterceptor().apply {
    level = if (BuildConfig.DEBUG) {
        HttpLoggingInterceptor.Level.BODY
    } else {
        HttpLoggingInterceptor.Level.NONE
    }
}
```

## Contributing

1. Follow Android development best practices
2. Use Material Design 3 guidelines
3. Write unit tests for ViewModels and repositories
4. Ensure proper error handling and user feedback
5. Maintain consistent code style with ktlint

## License

This project is part of the Music Server V2 project and follows the same license terms.
