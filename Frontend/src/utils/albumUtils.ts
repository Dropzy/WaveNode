import SHA256 from 'crypto-js/sha256';
import encHex from 'crypto-js/enc-hex';

// Utility function to generate album hash (matches backend implementation exactly)
export const generateAlbumHash = (albumName: string, artistName: string): string => {
  // Match backend exactly: fmt.Sprintf("%s|%s", name, artist) then SHA256
  const combined = `${albumName}|${artistName}`;
  
  // Use crypto-js SHA256 to match backend exactly
  const fullHash = SHA256(combined).toString(encHex);
  
  // Backend does: hex.EncodeToString(hash[:4]) - takes first 4 bytes of SHA256
  // This equals first 8 hex characters (4 bytes * 2 hex chars/byte)
  return fullHash.substring(0, 8);
}
