import { base64URLToBytes, bytesToBase64URL } from "./base64url";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

export type EncryptedItem = {
  bytes_base64: string;
  content_type: string;
  name: string;
};

type Envelope = {
  algorithm: "AES-GCM";
  ciphertext: string;
  iv: string;
  version: 1;
};

type Manifest = {
  items: EncryptedItem[];
  version: 1;
};

export async function encryptItems(
  items: EncryptedItem[],
): Promise<{ envelope: string; key: Uint8Array }> {
  const key = crypto.getRandomValues(new Uint8Array(32));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const plainText = encoder.encode(JSON.stringify({ version: 1, items } satisfies Manifest));
  const cryptoKey = await crypto.subtle.importKey("raw", key, "AES-GCM", false, ["encrypt"]);
  const cipherText = await crypto.subtle.encrypt({ name: "AES-GCM", iv }, cryptoKey, plainText);
  const envelope: Envelope = {
    version: 1,
    algorithm: "AES-GCM",
    iv: bytesToBase64URL(iv),
    ciphertext: bytesToBase64URL(new Uint8Array(cipherText)),
  };

  return { envelope: JSON.stringify(envelope), key };
}

export async function decryptItems(
  envelopeJSON: string,
  encodedKey: string,
): Promise<EncryptedItem[]> {
  const keyBytes = base64URLToBytes(encodedKey);
  if (keyBytes.length !== 32) {
    throw new Error("This URL has an invalid decryption key.");
  }

  const envelope = JSON.parse(envelopeJSON) as Envelope;
  if (envelope.version !== 1 || envelope.algorithm !== "AES-GCM") {
    throw new Error("Unsupported encrypted-share format.");
  }

  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    toArrayBuffer(keyBytes),
    "AES-GCM",
    false,
    ["decrypt"],
  );
  const plainText = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: toArrayBuffer(base64URLToBytes(envelope.iv)) },
    cryptoKey,
    toArrayBuffer(base64URLToBytes(envelope.ciphertext)),
  );
  const manifest = JSON.parse(decoder.decode(plainText)) as Manifest;
  if (manifest.version !== 1 || !Array.isArray(manifest.items)) {
    throw new Error("Invalid decrypted-share format.");
  }

  return manifest.items;
}

export function decodeText(bytes: Uint8Array): string {
  return decoder.decode(bytes);
}

export function toArrayBuffer(bytes: Uint8Array): ArrayBuffer {
  const copy = new Uint8Array(bytes.byteLength);
  copy.set(bytes);
  return copy.buffer;
}
