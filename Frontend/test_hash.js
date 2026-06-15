const SHA256 = require('crypto-js/sha256');
const encHex = require('crypto-js/enc-hex');

// Utility function to generate album hash (matches backend implementation exactly)
const generateAlbumHash = (albumName, artistName) => {
  // Match backend exactly: fmt.Sprintf("%s|%s", name, artist) then SHA256
  const combined = `${albumName}|${artistName}`;
  
  // Use crypto-js SHA256 to match backend exactly
  const fullHash = SHA256(combined).toString(encHex);
  
  // Backend does: hex.EncodeToString(hash[:4]) - takes first 4 bytes of SHA256
  // This equals first 8 hex characters (4 bytes * 2 hex chars/byte)
  return fullHash.substring(0, 8);
};

// Test with same data
console.log('Testing JavaScript hash generation:');
const albumName = 'Answers Vip / Activate Vip';
const artistName = 'Infrared';
const id = generateAlbumHash(albumName, artistName);
console.log(`Album: ${albumName}, Artist: ${artistName}, ID: ${id}`);

// Test another album
const albumName2 = 'Atlantis Ep';
const artistName2 = 'Infrared';
const id2 = generateAlbumHash(albumName2, artistName2);
console.log(`Album: ${albumName2}, Artist: ${artistName2}, ID: ${id2}`);
