import { bytesToBase64URL } from "../../shared/crypto/base64url";
import type { EncryptedItem } from "../../shared/crypto/share-crypto";
import type { DraftItem } from "./types";

const encoder = new TextEncoder();

export function draftItemsByteLength(items: DraftItem[]): number {
  return items.reduce(
    (total, item) => total + (item.file?.size ?? 0) + encoder.encode(item.text).byteLength,
    0,
  );
}

export async function collectItems(items: DraftItem[]): Promise<EncryptedItem[]> {
  const result: EncryptedItem[] = [];
  for (const [index, item] of items.entries()) {
    if (!item.file && item.text.length === 0) {
      continue;
    }

    const bytes = item.file
      ? new Uint8Array(await item.file.arrayBuffer())
      : encoder.encode(item.text);
    result.push({
      name: item.name.trim() || item.file?.name || `item-${index + 1}.txt`,
      content_type: item.file
        ? item.file.type || "application/octet-stream"
        : "text/plain;charset=utf-8",
      bytes_base64: bytesToBase64URL(bytes),
      ...(item.file && item.text.length > 0 ? { accompanying_text: item.text } : {}),
    });
  }

  return result;
}
