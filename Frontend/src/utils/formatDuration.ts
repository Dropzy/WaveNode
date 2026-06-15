/**
 * Formats duration in seconds to a human-readable string (MM:SS or H:MM:SS)
 */
export const formatDuration = (duration: number): string => {
  if (!duration || duration < 0) return '0:00';

  const hours = Math.floor(duration / 3600);
  const minutes = Math.floor((duration % 3600) / 60);
  const seconds = Math.floor(duration % 60);

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  }

  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};
