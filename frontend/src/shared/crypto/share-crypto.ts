import { base64URLToBytes, bytesToBase64URL } from "./base64url";

const encoder = new TextEncoder();
const decoder = new TextDecoder();
const aesGCMAuthenticationTagBytes = 16;
const aesGCMInitializationVectorBytes = 12;
const passwordSaltBytes = 16;
const passwordKDFIterations = 600_000;
const passwordVerifier = encoder.encode("sharelock-password-verifier-v1");
const passwordVerifierAdditionalData = encoder.encode("sharelock/password-verifier/v1");
const passwordPayloadAdditionalData = encoder.encode("sharelock/share-payload/v2");

export type EncryptedItem = {
  bytes_base64: string;
  content_type: string;
  name: string;
};

export type EncryptedShare = {
  accessEnvelope?: string;
  envelope: string;
  key: Uint8Array;
};

type Envelope = {
  algorithm: "AES-GCM";
  ciphertext: string;
  iv: string;
};

type AccessEnvelope = Envelope & {
  kdf: {
    hash: "SHA-256";
    iterations: number;
    name: "PBKDF2";
    salt: string;
  };
};

type Manifest = {
  items: EncryptedItem[];
  version: 1;
};

// encryptedShareByteLength calculates the total encrypted data size before encrypting it.
export function encryptedShareByteLength(items: EncryptedItem[], password?: string): number {
  const plainText = manifestBytes(items);
  const payloadEnvelope = JSON.stringify({
    algorithm: "AES-GCM",
    iv: "x".repeat(base64URLEncodedLength(aesGCMInitializationVectorBytes)),
    ciphertext: "x".repeat(
      base64URLEncodedLength(plainText.byteLength + aesGCMAuthenticationTagBytes),
    ),
  } satisfies Envelope);

  if (password === undefined) {
    return encoder.encode(payloadEnvelope).byteLength;
  }

  const accessEnvelope: AccessEnvelope = {
    algorithm: "AES-GCM",
    iv: "x".repeat(base64URLEncodedLength(aesGCMInitializationVectorBytes)),
    ciphertext: "x".repeat(
      base64URLEncodedLength(passwordVerifier.byteLength + aesGCMAuthenticationTagBytes),
    ),
    kdf: {
      name: "PBKDF2",
      hash: "SHA-256",
      iterations: passwordKDFIterations,
      salt: "x".repeat(base64URLEncodedLength(passwordSaltBytes)),
    },
  };

  return (
    encoder.encode(payloadEnvelope).byteLength +
    encoder.encode(JSON.stringify(accessEnvelope)).byteLength
  );
}

export async function encryptItems(
  items: EncryptedItem[],
  password?: string,
): Promise<EncryptedShare> {
  const key = crypto.getRandomValues(new Uint8Array(32));
  if (password === undefined) {
    return encryptWithoutPassword(items, key);
  }
  if (password.length === 0) {
    throw new Error("A password-protected share requires a password.");
  }

  const salt = crypto.getRandomValues(new Uint8Array(passwordSaltBytes));
  const cryptoKey = await derivePasswordKey(key, password, salt);
  const accessEnvelope = await encryptAccessEnvelope(cryptoKey, salt);
  const envelope = await encryptPasswordProtectedPayload(items, cryptoKey);

  return { envelope, accessEnvelope, key };
}

export async function decryptItems(
  envelopeJSON: string,
  encodedKey: string,
): Promise<EncryptedItem[]> {
  const keyBytes = decodeShareKey(encodedKey);
  const envelope = parseEnvelope(envelopeJSON);
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    toArrayBuffer(keyBytes),
    "AES-GCM",
    false,
    ["decrypt"],
  );

  return decryptManifest(envelope, cryptoKey);
}

export async function verifyPassword(
  accessEnvelopeJSON: string,
  encodedKey: string,
  password: string,
): Promise<CryptoKey> {
  if (password.length === 0) {
    throw new Error("Enter the password for this share.");
  }

  const accessEnvelope = parseAccessEnvelope(accessEnvelopeJSON);
  const salt = decodeFixedLengthBase64URL(
    accessEnvelope.kdf.salt,
    passwordSaltBytes,
    "Invalid password-protected share format.",
  );
  const cryptoKey = await derivePasswordKey(decodeShareKey(encodedKey), password, salt);

  try {
    const plainText = new Uint8Array(
      await crypto.subtle.decrypt(
        {
          name: "AES-GCM",
          iv: toArrayBuffer(decodeIV(accessEnvelope.iv)),
          additionalData: toArrayBuffer(passwordVerifierAdditionalData),
        },
        cryptoKey,
        toArrayBuffer(decodeCiphertext(accessEnvelope.ciphertext)),
      ),
    );
    if (!bytesEqual(plainText, passwordVerifier)) {
      throw new Error("Password verifier mismatch.");
    }
  } catch {
    throw new Error("The password is incorrect or the password verifier is damaged.");
  }

  return cryptoKey;
}

export async function decryptPasswordProtectedItems(
  envelopeJSON: string,
  cryptoKey: CryptoKey,
): Promise<EncryptedItem[]> {
  return decryptManifest(parseEnvelope(envelopeJSON), cryptoKey, passwordPayloadAdditionalData);
}

export function decodeText(bytes: Uint8Array): string {
  return decoder.decode(bytes);
}

export function toArrayBuffer(bytes: Uint8Array): ArrayBuffer {
  const copy = new Uint8Array(bytes.byteLength);
  copy.set(bytes);
  return copy.buffer;
}

async function encryptWithoutPassword(
  items: EncryptedItem[],
  key: Uint8Array,
): Promise<EncryptedShare> {
  const iv = crypto.getRandomValues(new Uint8Array(aesGCMInitializationVectorBytes));
  const cryptoKey = await crypto.subtle.importKey("raw", toArrayBuffer(key), "AES-GCM", false, [
    "encrypt",
  ]);
  const cipherText = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: toArrayBuffer(iv) },
    cryptoKey,
    toArrayBuffer(manifestBytes(items)),
  );
  const envelope: Envelope = {
    algorithm: "AES-GCM",
    iv: bytesToBase64URL(iv),
    ciphertext: bytesToBase64URL(new Uint8Array(cipherText)),
  };

  return { envelope: JSON.stringify(envelope), key };
}

async function encryptPasswordProtectedPayload(
  items: EncryptedItem[],
  cryptoKey: CryptoKey,
): Promise<string> {
  const iv = crypto.getRandomValues(new Uint8Array(aesGCMInitializationVectorBytes));
  const cipherText = await crypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv: toArrayBuffer(iv),
      additionalData: passwordPayloadAdditionalData,
    },
    cryptoKey,
    toArrayBuffer(manifestBytes(items)),
  );
  const envelope: Envelope = {
    algorithm: "AES-GCM",
    iv: bytesToBase64URL(iv),
    ciphertext: bytesToBase64URL(new Uint8Array(cipherText)),
  };

  return JSON.stringify(envelope);
}

async function encryptAccessEnvelope(cryptoKey: CryptoKey, salt: Uint8Array): Promise<string> {
  const iv = crypto.getRandomValues(new Uint8Array(aesGCMInitializationVectorBytes));
  const cipherText = await crypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv: toArrayBuffer(iv),
      additionalData: passwordVerifierAdditionalData,
    },
    cryptoKey,
    toArrayBuffer(passwordVerifier),
  );
  const accessEnvelope: AccessEnvelope = {
    algorithm: "AES-GCM",
    iv: bytesToBase64URL(iv),
    ciphertext: bytesToBase64URL(new Uint8Array(cipherText)),
    kdf: {
      name: "PBKDF2",
      hash: "SHA-256",
      iterations: passwordKDFIterations,
      salt: bytesToBase64URL(salt),
    },
  };

  return JSON.stringify(accessEnvelope);
}

async function derivePasswordKey(
  shareKey: Uint8Array,
  password: string,
  salt: Uint8Array,
): Promise<CryptoKey> {
  const keyMaterial = concatenateBytes(shareKey, encoder.encode(password));
  const baseKey = await crypto.subtle.importKey(
    "raw",
    toArrayBuffer(keyMaterial),
    "PBKDF2",
    false,
    ["deriveKey"],
  );

  return crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      hash: "SHA-256",
      salt: toArrayBuffer(salt),
      iterations: passwordKDFIterations,
    },
    baseKey,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

async function decryptManifest(
  envelope: Envelope,
  cryptoKey: CryptoKey,
  additionalData?: Uint8Array,
): Promise<EncryptedItem[]> {
  let plainText: ArrayBuffer;
  try {
    plainText = await crypto.subtle.decrypt(
      {
        name: "AES-GCM",
        iv: toArrayBuffer(decodeIV(envelope.iv)),
        ...(additionalData === undefined ? {} : { additionalData: toArrayBuffer(additionalData) }),
      },
      cryptoKey,
      toArrayBuffer(decodeCiphertext(envelope.ciphertext)),
    );
  } catch {
    throw new Error("The decryption key is invalid or the encrypted payload is damaged.");
  }

  const manifest = parseJSON(decoder.decode(plainText), "Invalid decrypted-share format.");
  if (!isManifest(manifest)) {
    throw new Error("Invalid decrypted-share format.");
  }

  return manifest.items;
}

function parseEnvelope(envelopeJSON: string): Envelope {
  const envelope = parseJSON(envelopeJSON, "Unsupported encrypted-share format.");
  if (!isEnvelope(envelope)) {
    throw new Error("Unsupported encrypted-share format.");
  }

  return envelope;
}

function parseAccessEnvelope(envelopeJSON: string): AccessEnvelope {
  const envelope = parseJSON(envelopeJSON, "Invalid password-protected share format.");
  if (
    !isRecord(envelope) ||
    envelope.algorithm !== "AES-GCM" ||
    typeof envelope.iv !== "string" ||
    typeof envelope.ciphertext !== "string" ||
    !isRecord(envelope.kdf) ||
    envelope.kdf.name !== "PBKDF2" ||
    envelope.kdf.hash !== "SHA-256" ||
    envelope.kdf.iterations !== passwordKDFIterations ||
    typeof envelope.kdf.salt !== "string"
  ) {
    throw new Error("Invalid password-protected share format.");
  }

  return envelope as AccessEnvelope;
}

function decodeShareKey(encodedKey: string): Uint8Array {
  return decodeFixedLengthBase64URL(encodedKey, 32, "This URL has an invalid decryption key.");
}

function decodeIV(encodedIV: string): Uint8Array {
  return decodeFixedLengthBase64URL(
    encodedIV,
    aesGCMInitializationVectorBytes,
    "Invalid encrypted-share format.",
  );
}

function decodeCiphertext(encodedCiphertext: string): Uint8Array {
  try {
    const cipherText = base64URLToBytes(encodedCiphertext);
    if (cipherText.length < aesGCMAuthenticationTagBytes) {
      throw new Error("ciphertext too short");
    }

    return cipherText;
  } catch {
    throw new Error("Invalid encrypted-share format.");
  }
}

function decodeFixedLengthBase64URL(encoded: string, length: number, message: string): Uint8Array {
  try {
    const decoded = base64URLToBytes(encoded);
    if (decoded.length !== length) {
      throw new Error("invalid length");
    }

    return decoded;
  } catch {
    throw new Error(message);
  }
}

function manifestBytes(items: EncryptedItem[]): Uint8Array {
  return encoder.encode(JSON.stringify({ version: 1, items } satisfies Manifest));
}

function parseJSON(value: string, message: string): unknown {
  try {
    return JSON.parse(value);
  } catch {
    throw new Error(message);
  }
}

function isEnvelope(value: unknown): value is Envelope {
  return (
    isRecord(value) &&
    value.algorithm === "AES-GCM" &&
    typeof value.iv === "string" &&
    typeof value.ciphertext === "string"
  );
}

function isManifest(value: unknown): value is Manifest {
  return isRecord(value) && value.version === 1 && Array.isArray(value.items);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function concatenateBytes(left: Uint8Array, right: Uint8Array): Uint8Array {
  const result = new Uint8Array(left.byteLength + right.byteLength);
  result.set(left);
  result.set(right, left.byteLength);
  return result;
}

function bytesEqual(left: Uint8Array, right: Uint8Array): boolean {
  if (left.byteLength !== right.byteLength) {
    return false;
  }

  let different = 0;
  for (let index = 0; index < left.byteLength; index += 1) {
    different |= left[index] ^ right[index];
  }

  return different === 0;
}

function base64URLEncodedLength(byteLength: number): number {
  return Math.ceil((byteLength * 8) / 6);
}
