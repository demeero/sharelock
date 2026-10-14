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

export type EncryptedFile = {
  bytes_base64: string;
  content_type: string;
  name: string;
};

export type PayloadBlock = {
  file?: EncryptedFile;
  text?: string;
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

type LegacyManifest = {
  items: EncryptedFile[];
  version: 1;
};

type Version2Manifest = {
  blocks: PayloadBlock[];
  version: 2;
};

// encryptedShareByteLength calculates the total encrypted data size before encrypting it.
export function encryptedShareByteLength(payloadBlocks: PayloadBlock[], password?: string): number {
  const plainText = manifestBytes(payloadBlocks);
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

export async function encryptPayloadBlocks(
  payloadBlocks: PayloadBlock[],
  password?: string,
): Promise<EncryptedShare> {
  const normalizedPayloadBlocks = normalizePayloadBlocks(payloadBlocks, "Invalid payload block.");
  const key = crypto.getRandomValues(new Uint8Array(32));
  if (password === undefined) {
    return encryptWithoutPassword(normalizedPayloadBlocks, key);
  }
  if (password.length === 0) {
    throw new Error("A password-protected share requires a password.");
  }

  const salt = crypto.getRandomValues(new Uint8Array(passwordSaltBytes));
  const cryptoKey = await derivePasswordKey(key, password, salt);
  const accessEnvelope = await encryptAccessEnvelope(cryptoKey, salt);
  const envelope = await encryptPasswordProtectedPayload(normalizedPayloadBlocks, cryptoKey);

  return { envelope, accessEnvelope, key };
}

export async function decryptPayloadBlocks(
  envelopeJSON: string,
  encodedKey: string,
): Promise<PayloadBlock[]> {
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

export async function decryptPasswordProtectedPayloadBlocks(
  envelopeJSON: string,
  cryptoKey: CryptoKey,
): Promise<PayloadBlock[]> {
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
  payloadBlocks: PayloadBlock[],
  key: Uint8Array,
): Promise<EncryptedShare> {
  const iv = crypto.getRandomValues(new Uint8Array(aesGCMInitializationVectorBytes));
  const cryptoKey = await crypto.subtle.importKey("raw", toArrayBuffer(key), "AES-GCM", false, [
    "encrypt",
  ]);
  const cipherText = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: toArrayBuffer(iv) },
    cryptoKey,
    toArrayBuffer(manifestBytes(payloadBlocks)),
  );
  const envelope: Envelope = {
    algorithm: "AES-GCM",
    iv: bytesToBase64URL(iv),
    ciphertext: bytesToBase64URL(new Uint8Array(cipherText)),
  };

  return { envelope: JSON.stringify(envelope), key };
}

async function encryptPasswordProtectedPayload(
  payloadBlocks: PayloadBlock[],
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
    toArrayBuffer(manifestBytes(payloadBlocks)),
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
): Promise<PayloadBlock[]> {
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

  return parseManifest(decoder.decode(plainText));
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

function manifestBytes(payloadBlocks: PayloadBlock[]): Uint8Array {
  const blocks = normalizePayloadBlocks(payloadBlocks, "Invalid payload block.");
  return encoder.encode(JSON.stringify({ version: 2, blocks } satisfies Version2Manifest));
}

function parseManifest(value: string): PayloadBlock[] {
  const manifest = parseJSON(value, "Invalid decrypted-share format.");
  if (isLegacyManifest(manifest)) {
    return manifest.items.map((file) => ({ file }));
  }
  if (!isVersion2Manifest(manifest)) {
    throw new Error("Invalid decrypted-share format.");
  }

  return normalizePayloadBlocks(manifest.blocks, "Invalid decrypted-share format.");
}

function normalizePayloadBlocks(payloadBlocks: PayloadBlock[], message: string): PayloadBlock[] {
  if (!Array.isArray(payloadBlocks)) {
    throw new Error(message);
  }

  return payloadBlocks.map((payloadBlock) => normalizePayloadBlock(payloadBlock, message));
}

function normalizePayloadBlock(payloadBlock: PayloadBlock, message: string): PayloadBlock {
  if (!isPayloadBlockShape(payloadBlock)) {
    throw new Error(message);
  }

  const text = payloadBlock.text?.trim().length === 0 ? undefined : payloadBlock.text;
  if (payloadBlock.file === undefined && text === undefined) {
    throw new Error(message);
  }

  return {
    ...(payloadBlock.file === undefined ? {} : { file: payloadBlock.file }),
    ...(text === undefined ? {} : { text }),
  };
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

function isLegacyManifest(value: unknown): value is LegacyManifest {
  return (
    isRecord(value) &&
    value.version === 1 &&
    Array.isArray(value.items) &&
    value.items.every(isEncryptedFile)
  );
}

function isVersion2Manifest(value: unknown): value is Version2Manifest {
  return (
    isRecord(value) &&
    value.version === 2 &&
    Array.isArray(value.blocks) &&
    value.blocks.every(isPayloadBlockShape)
  );
}

function isPayloadBlockShape(value: unknown): value is PayloadBlock {
  if (!isRecord(value)) {
    return false;
  }

  return (
    (value.file === undefined || isEncryptedFile(value.file)) &&
    (value.text === undefined || typeof value.text === "string")
  );
}

function isEncryptedFile(value: unknown): value is EncryptedFile {
  return (
    isRecord(value) &&
    typeof value.name === "string" &&
    typeof value.content_type === "string" &&
    typeof value.bytes_base64 === "string" &&
    isBase64URL(value.bytes_base64)
  );
}

function isBase64URL(value: string): boolean {
  try {
    return bytesToBase64URL(base64URLToBytes(value)) === value;
  } catch {
    return false;
  }
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
