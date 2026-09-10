import { describe, expect, it } from "vitest";

import { bytesToBase64URL } from "./base64url";
import {
  decryptItems,
  decryptPasswordProtectedItems,
  encryptedShareByteLength,
  encryptItems,
  verifyPassword,
  type EncryptedItem,
} from "./share-crypto";

const items: EncryptedItem[] = [
  {
    name: "secret.txt",
    content_type: "text/plain;charset=utf-8",
    bytes_base64: bytesToBase64URL(new TextEncoder().encode("correct horse battery staple")),
  },
];

describe("share crypto", () => {
  it("keeps shares without a password readable with only the URL key", async () => {
    const share = await encryptItems(items);

    await expect(decryptItems(share.envelope, bytesToBase64URL(share.key))).resolves.toEqual(items);
  });

  it("requires both the URL key and password for a password-protected share", async () => {
    const share = await encryptItems(items, "separate password");

    if (!share.accessEnvelope) {
      throw new Error("expected a password-protected share");
    }
    const cryptoKey = await verifyPassword(
      share.accessEnvelope,
      bytesToBase64URL(share.key),
      "separate password",
    );
    await expect(decryptPasswordProtectedItems(share.envelope, cryptoKey)).resolves.toEqual(items);
  });

  it("rejects an incorrect password without needing the payload envelope", async () => {
    const share = await encryptItems(items, "separate password");
    if (!share.accessEnvelope) {
      throw new Error("expected a password-protected share");
    }

    await expect(
      verifyPassword(share.accessEnvelope, bytesToBase64URL(share.key), "incorrect password"),
    ).rejects.toThrow("password is incorrect");
  });

  it("rejects malformed password verifier metadata", async () => {
    const share = await encryptItems(items, "separate password");

    await expect(
      verifyPassword('{"algorithm":"AES-GCM"}', bytesToBase64URL(share.key), "separate password"),
    ).rejects.toThrow("Invalid password-protected share format");
  });

  it("includes the verifier when estimating password-protected share size", async () => {
    const share = await encryptItems(items, "separate password");
    const actualBytes =
      new TextEncoder().encode(share.envelope).byteLength +
      new TextEncoder().encode(share.accessEnvelope ?? "").byteLength;

    expect(encryptedShareByteLength(items, "separate password")).toBe(actualBytes);
  });
});
