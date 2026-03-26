export interface EncryptionKeyConfig {
  /** Current key ID (for key rotation) */
  currentKeyId: string;
  /** Map of key ID to key material (hex or base64-encoded 256-bit keys) */
  keys: Record<string, string>;
}

export interface EncryptedPayload {
  /** Key ID used for encryption */
  kid: string;
  /** Initialization vector (base64) */
  iv: string;
  /** Encrypted data (base64) */
  data: string;
  /** Authentication tag (base64) */
  tag: string;
  /** Format version */
  v: number;
}
