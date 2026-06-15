// Utility function to generate artist hash (same as backend)
export const generateArtistHash = (artistName: string): string => {
  const normalizedString = artistName.toLowerCase().trim()
  let hash = 0
  for (let i = 0; i < normalizedString.length; i++) {
    const char = normalizedString.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash // Convert to 32bit integer
  }
  // Convert to unsigned 32-bit integer and format as 8-character hex with leading zeros
  const unsignedHash = hash >>> 0 // Convert to unsigned 32-bit
  const hexHash = unsignedHash.toString(16)
  return hexHash.padStart(8, '0')
}
