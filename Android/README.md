# WaveNode Android Player

Fresh native Android client for WaveNode.

## Current Milestone

- Connect to a WaveNode server by URL
- Sign in with an existing WaveNode account
- Load the track library
- Stream tracks with Media3/ExoPlayer
- Show a persistent mini-player

## Build

Open the `Android` folder in Android Studio, or build from the command line:

```powershell
.\gradlew.bat assembleDebug
```

The debug APK is written to:

```text
Android/app/build/outputs/apk/debug/app-debug.apk
```

## Local Network

For a local WaveNode server, use the LAN address shown by the server machine, for example:

```text
http://192.168.1.70:8080
```

Cleartext HTTP is enabled for local self-hosted testing. HTTPS should be used for remote access.

## Next Milestones

- Queue controls and seek/progress display
- Background media session and lock-screen controls
- Album, artist, playlist, and search screens
- Offline downloads using the existing WaveNode download endpoint
- Android Auto support
