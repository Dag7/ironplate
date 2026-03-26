import { createCipheriv, createDecipheriv, randomBytes } from 'crypto';
import type { EncryptionKeyConfig, EncryptedPayload } from './types';

const ALGORITHM = 'aes-256-gcm';
const IV_LENGTH = 16;
const TAG_LENGTH = 16;
const FORMAT_VERSION = 1;

/**
 * Envelope encryption service with key rotation support.
 *
 * Uses AES-256-GCM for authenticated encryption. Supports multiple keys
 * for zero-downtime key rotation: encrypt with the current key, decrypt
 * with any known key.
 *
 * @example
 * ```ts
 * const enc = new EncryptionService({
 *   currentKeyId: 'key-2024',
 *   keys: {
 *     'key-2023': process.env.ENCRYPTION_KEY_2023!,
 *     'key-2024': process.env.ENCRYPTION_KEY_2024!,
 *   },
 * });
 *
 * const encrypted = enc.encrypt('sensitive data');
 * const decrypted = enc.decrypt(encrypted); // 'sensitive data'
 * ```
 */
export class EncryptionService {
  private readonly currentKeyId: string;
  private readonly keys: Map<string, Buffer>;

  constructor(config: EncryptionKeyConfig) {
    this.currentKeyId = config.currentKeyId;
    this.keys = new Map();

    for (const [id, material] of Object.entries(config.keys)) {
      const key = Buffer.from(material, 'hex');
      if (key.length !== 32) {
        throw new Error(`Key "${id}" must be 256 bits (32 bytes), got ${key.length} bytes`);
      }
      this.keys.set(id, key);
    }

    if (!this.keys.has(this.currentKeyId)) {
      throw new Error(`Current key "${this.currentKeyId}" not found in key set`);
    }
  }

  /**
   * Encrypt a string value using the current key.
   */
  encrypt(plaintext: string): EncryptedPayload {
    const key = this.keys.get(this.currentKeyId)!;
    const iv = randomBytes(IV_LENGTH);

    const cipher = createCipheriv(ALGORITHM, key, iv, { authTagLength: TAG_LENGTH });
    const encrypted = Buffer.concat([cipher.update(plaintext, 'utf8'), cipher.final()]);
    const tag = cipher.getAuthTag();

    return {
      kid: this.currentKeyId,
      iv: iv.toString('base64'),
      data: encrypted.toString('base64'),
      tag: tag.toString('base64'),
      v: FORMAT_VERSION,
    };
  }

  /**
   * Decrypt an encrypted payload using the key specified in the payload.
   */
  decrypt(payload: EncryptedPayload): string {
    const key = this.keys.get(payload.kid);
    if (!key) {
      throw new Error(`Unknown key ID: "${payload.kid}"`);
    }

    const iv = Buffer.from(payload.iv, 'base64');
    const data = Buffer.from(payload.data, 'base64');
    const tag = Buffer.from(payload.tag, 'base64');

    const decipher = createDecipheriv(ALGORITHM, key, iv, { authTagLength: TAG_LENGTH });
    decipher.setAuthTag(tag);

    return Buffer.concat([decipher.update(data), decipher.final()]).toString('utf8');
  }

  /**
   * Encrypt a value and return it as a JSON string (for database storage).
   */
  encryptToString(plaintext: string): string {
    return JSON.stringify(this.encrypt(plaintext));
  }

  /**
   * Decrypt a JSON string payload.
   */
  decryptFromString(encrypted: string): string {
    const payload: EncryptedPayload = JSON.parse(encrypted);
    return this.decrypt(payload);
  }

  /**
   * Check if a payload was encrypted with the current key.
   * Returns false if re-encryption with the new key is needed.
   */
  isCurrentKey(payload: EncryptedPayload): boolean {
    return payload.kid === this.currentKeyId;
  }

  /**
   * Re-encrypt a payload with the current key (for key rotation).
   */
  rotate(payload: EncryptedPayload): EncryptedPayload {
    const plaintext = this.decrypt(payload);
    return this.encrypt(plaintext);
  }

  /**
   * Generate a random 256-bit key suitable for use with this service.
   */
  static generateKey(): string {
    return randomBytes(32).toString('hex');
  }
}
