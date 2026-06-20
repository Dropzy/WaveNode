package org.wavenode.player.ui

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

val WaveBackground = Color(0xFF070B12)
val WaveSurface = Color(0xFF151A22)
val WaveSurfaceRaised = Color(0xFF202631)
val WaveAccent = Color(0xFF63A8FF)
val WaveText = Color(0xFFFFFFFF)
val WaveSubtle = Color(0xFFC3CAD7)

private val WaveNodeColors: ColorScheme = darkColorScheme(
    primary = WaveAccent,
    onPrimary = Color(0xFF061321),
    background = WaveBackground,
    onBackground = WaveText,
    surface = WaveSurface,
    onSurface = WaveText,
    surfaceVariant = WaveSurfaceRaised,
    onSurfaceVariant = WaveSubtle,
)

@Composable
fun WaveNodeTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = WaveNodeColors,
        content = content,
    )
}
