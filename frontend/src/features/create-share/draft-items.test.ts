import { describe, expect, it } from "vitest";

import { base64URLToBytes } from "../../shared/crypto/base64url";
import { collectItems, draftItemsByteLength } from "./draft-items";
import type { DraftItem } from "./types";

function draft(overrides: Partial<DraftItem> = {}): DraftItem {
  return { id: 1, name: "", text: "", file: null, ...overrides };
}

describe("draft share items", () => {
  it("keeps file bytes and accompanying text together without changing either", async () => {
    const bytes = new Uint8Array([0, 255, 128, 42]);
    const text = '  Інструкція 🔐\n<script>alert("secret")</script>\n**literal**  ';
    const file = new File([bytes], "secret.bin", { type: "application/octet-stream" });

    const result = await collectItems([draft({ file, text })]);

    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("secret.bin");
    expect(result[0].content_type).toBe("application/octet-stream");
    expect(base64URLToBytes(result[0].bytes_base64)).toEqual(bytes);
    expect(result[0].accompanying_text).toBe(text);
  });

  it("preserves text-only, file-only and mixed items and skips empty cards", async () => {
    const file = new File(["original file"], "config.txt", { type: "text/plain" });

    const result = await collectItems([
      draft(),
      draft({ text: "text only" }),
      draft({ file }),
      draft({ file, text: "file instructions", name: "  custom.txt  " }),
    ]);

    expect(result).toHaveLength(3);
    expect(new TextDecoder().decode(base64URLToBytes(result[0].bytes_base64))).toBe("text only");
    expect(result[0]).not.toHaveProperty("accompanying_text");
    expect(result[1].name).toBe("config.txt");
    expect(result[1]).not.toHaveProperty("accompanying_text");
    expect(result[2].name).toBe("custom.txt");
    expect(result[2].accompanying_text).toBe("file instructions");
  });

  it("preserves whitespace-only accompanying and standalone text", async () => {
    const text = " \n\t  ";
    const result = await collectItems([
      draft({ file: new File(["file"], "secret.txt"), text }),
      draft({ text }),
    ]);

    expect(result[0].accompanying_text).toBe(text);
    expect(new TextDecoder().decode(base64URLToBytes(result[1].bytes_base64))).toBe(text);
  });

  it("retains zero-byte files and uses a binary fallback for files without a MIME type", async () => {
    const result = await collectItems([draft({ file: new File([], "empty.bin"), text: "note" })]);

    expect(result).toHaveLength(1);
    expect(result[0].bytes_base64).toBe("");
    expect(result[0].content_type).toBe("application/octet-stream");
    expect(result[0].accompanying_text).toBe("note");
  });

  it("adds file sizes and UTF-8 text sizes across all cards", () => {
    const file = new File([new Uint8Array([0, 255, 128, 42])], "secret.bin");
    const text = "Привіт 🔐\n";

    expect(
      draftItemsByteLength([draft({ file, text }), draft({ file }), draft({ text }), draft()]),
    ).toBe(2 * file.size + 2 * new TextEncoder().encode(text).byteLength);
  });
});
