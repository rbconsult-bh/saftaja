import { createCipheriv, randomBytes } from 'crypto';

const NONCE_SIZE = 12; // GCM nonce size

export function encrypt(plaintext: Buffer, key: Buffer): Buffer {
  if (key.length !== 32) {
    throw new Error('Encryption key must be 32 bytes');
  }

  const nonce = randomBytes(NONCE_SIZE);
  const cipher = createCipheriv('aes-256-gcm', key, nonce);

  const encrypted = Buffer.concat([cipher.update(plaintext), cipher.final()]);
  const authTag = cipher.getAuthTag();

  // Format: nonce + ciphertext + authTag (matches Go's gcm.Seal)
  return Buffer.concat([nonce, encrypted, authTag]);
}

export function encryptJSON(data: object, key: Buffer): Buffer {
  const plaintext = Buffer.from(JSON.stringify(data), 'utf-8');
  return encrypt(plaintext, key);
}
