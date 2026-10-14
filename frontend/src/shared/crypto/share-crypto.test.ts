import { describe, expect, it } from "vitest";

import { base64URLToBytes, bytesToBase64URL } from "./base64url";
import {
  decryptPasswordProtectedPayloadBlocks,
  decryptPayloadBlocks,
  encryptedShareByteLength,
  encryptPayloadBlocks,
  toArrayBuffer,
  verifyPassword,
  type EncryptedFile,
  type PayloadBlock,
} from "./share-crypto";

const encoder = new TextEncoder();

const secretFile: EncryptedFile = {
  name: "secret.txt",
  content_type: "text/plain;charset=utf-8",
  bytes_base64: bytesToBase64URL(encoder.encode("correct horse battery staple")),
};

const payloadBlocks: PayloadBlock[] = [{ file: secretFile }, { text: "Read this first." }];

describe("share crypto", () => {
  it("creates version 2 payload blocks for file-only and text-note-only shares", async () => {
    const share = await encryptPayloadBlocks(payloadBlocks);

    await expect(
      decryptPayloadBlocks(share.envelope, bytesToBase64URL(share.key)),
    ).resolves.toEqual(payloadBlocks);

    await expect(decryptUnprotectedManifest(share.envelope, share.key)).resolves.toEqual({
      version: 2,
      blocks: payloadBlocks,
    });
  });

  it("requires both the URL key and password for version 2 payload blocks", async () => {
    const share = await encryptPayloadBlocks(payloadBlocks, "separate password");

    if (!share.accessEnvelope) {
      throw new Error("expected a password-protected share");
    }
    const cryptoKey = await verifyPassword(
      share.accessEnvelope,
      bytesToBase64URL(share.key),
      "separate password",
    );
    await expect(decryptPasswordProtectedPayloadBlocks(share.envelope, cryptoKey)).resolves.toEqual(
      payloadBlocks,
    );
  });

  it("rejects an incorrect password without needing the payload envelope", async () => {
    const share = await encryptPayloadBlocks(payloadBlocks, "separate password");
    if (!share.accessEnvelope) {
      throw new Error("expected a password-protected share");
    }

    await expect(
      verifyPassword(share.accessEnvelope, bytesToBase64URL(share.key), "incorrect password"),
    ).rejects.toThrow("password is incorrect");
  });

  it("rejects malformed password verifier metadata", async () => {
    const share = await encryptPayloadBlocks(payloadBlocks, "separate password");

    await expect(
      verifyPassword('{"algorithm":"AES-GCM"}', bytesToBase64URL(share.key), "separate password"),
    ).rejects.toThrow("Invalid password-protected share format");
  });

  it("opens every legacy item as its own payload block", async () => {
    const legacyFile: EncryptedFile = {
      name: "legacy.txt",
      content_type: "text/plain;charset=utf-8",
      bytes_base64: bytesToBase64URL(encoder.encode("legacy text")),
    };
    const legacyManifest = { version: 1, items: [secretFile, legacyFile] };
    const key = new Uint8Array(32).fill(7);
    const envelope = await encryptUnprotectedManifest(legacyManifest, key);

    await expect(decryptPayloadBlocks(envelope, bytesToBase64URL(key))).resolves.toEqual([
      { file: secretFile },
      { file: legacyFile },
    ]);
  });

  it("rejects empty version 2 payload blocks during encryption and decryption", async () => {
    await expect(encryptPayloadBlocks([{ text: " \n\t " }])).rejects.toThrow(
      "Invalid payload block",
    );

    const key = new Uint8Array(32).fill(8);
    const malformedEnvelope = await encryptUnprotectedManifest(
      {
        version: 2,
        blocks: [
          {
            file: {
              name: "malformed.txt",
              content_type: "text/plain;charset=utf-8",
              bytes_base64: "not-valid-base64url!",
            },
          },
        ],
      },
      key,
    );

    await expect(decryptPayloadBlocks(malformedEnvelope, bytesToBase64URL(key))).rejects.toThrow(
      "Invalid decrypted-share format",
    );
  });

  it("includes version 2 payload blocks and the verifier when estimating password-protected share size", async () => {
    const share = await encryptPayloadBlocks(payloadBlocks, "separate password");
    const actualBytes =
      encoder.encode(share.envelope).byteLength +
      encoder.encode(share.accessEnvelope ?? "").byteLength;

    expect(encryptedShareByteLength(payloadBlocks, "separate password")).toBe(actualBytes);
  });
});

async function decryptUnprotectedManifest(envelopeJSON: string, key: Uint8Array): Promise<unknown> {
  const envelope: { ciphertext: string; iv: string } = JSON.parse(envelopeJSON);
  const cryptoKey = await crypto.subtle.importKey("raw", toArrayBuffer(key), "AES-GCM", false, [
    "decrypt",
  ]);
  const plainText = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: toArrayBuffer(base64URLToBytes(envelope.iv)),
    },
    cryptoKey,
    toArrayBuffer(base64URLToBytes(envelope.ciphertext)),
  );

  return JSON.parse(new TextDecoder().decode(plainText));
}

async function encryptUnprotectedManifest(manifest: unknown, key: Uint8Array): Promise<string> {
  const iv = new Uint8Array(12).fill(9);
  const cryptoKey = await crypto.subtle.importKey("raw", toArrayBuffer(key), "AES-GCM", false, [
    "encrypt",
  ]);
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: toArrayBuffer(iv) },
    cryptoKey,
    encoder.encode(JSON.stringify(manifest)),
  );

  return JSON.stringify({
    algorithm: "AES-GCM",
    iv: bytesToBase64URL(iv),
    ciphertext: bytesToBase64URL(new Uint8Array(ciphertext)),
  });
}
