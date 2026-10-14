import { describe, expect, it } from "vitest";

import { bytesToBase64URL } from "./base64url";
import legacyShares from "./testdata/legacy-shares.json";
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
  it("opens fixed shares created by the pre-accompanying-text client", async () => {
    await expect(
      decryptItems(legacyShares.plain.envelope, legacyShares.plain.key),
    ).resolves.toEqual(legacyShares.items);
  });

  it("opens fixed password-protected shares created by the pre-accompanying-text client", async () => {
    const share = legacyShares.protected;
    const key = await verifyPassword(share.accessEnvelope, share.key, share.password);

    await expect(decryptPasswordProtectedItems(share.envelope, key)).resolves.toEqual(
      legacyShares.items,
    );
  });

  it.each([undefined, "separate password"])(
    "encrypts mixed items and counts their complete encrypted size (password: %s)",
    async (password) => {
      const mixedItems: EncryptedItem[] = [
        ...items,
        {
          name: 'binary-🔐".bin',
          content_type: "application/octet-stream",
          bytes_base64: bytesToBase64URL(new Uint8Array([0, 255, 128, 42])),
          accompanying_text: '  Інструкція 🔐\n"quotes" \\ slash\t<script>literal</script>  ',
        },
        { ...items[0], name: "file-only.txt" },
        { ...items[0], name: "whitespace.txt", accompanying_text: " \n\t " },
      ];

      const share = await encryptItems(mixedItems, password);
      const encodedKey = bytesToBase64URL(share.key);
      const decrypted =
        share.accessEnvelope === undefined
          ? await decryptItems(share.envelope, encodedKey)
          : await decryptPasswordProtectedItems(
              share.envelope,
              await verifyPassword(share.accessEnvelope, encodedKey, password!),
            );

      expect(decrypted).toEqual(mixedItems);
      expect(encryptedShareByteLength(mixedItems, password)).toBe(
        new TextEncoder().encode(share.envelope).byteLength +
          new TextEncoder().encode(share.accessEnvelope ?? "").byteLength,
      );
      expect(encryptedShareByteLength(mixedItems, password)).toBeGreaterThan(
        encryptedShareByteLength(
          mixedItems.map((item) => ({
            name: item.name,
            content_type: item.content_type,
            bytes_base64: item.bytes_base64,
          })),
          password,
        ),
      );
      expect(share.envelope).not.toContain("Інструкція");
      expect(share.envelope).not.toContain("accompanying_text");
      expect(share.envelope).not.toContain("binary-");
    },
  );

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
