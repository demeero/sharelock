<script lang="ts">
  import { base64URLToBytes } from "../../shared/crypto/base64url";
  import { decodeText, toArrayBuffer, type EncryptedItem } from "../../shared/crypto/share-crypto";
  import { formatBytes } from "../../shared/format/bytes";
  import Icon from "../../shared/ui/Icon.svelte";

  interface Props {
    item: EncryptedItem;
    index: number;
  }

  let { item, index }: Props = $props();
  let copyState = $state<"idle" | "copied" | "error">("idle");
  let accompanyingCopyState = $state<"idle" | "copied" | "error">("idle");
  let bytes = $derived(base64URLToBytes(item.bytes_base64));
  let isText = $derived(item.content_type.startsWith("text/"));
  let text = $derived(isText ? decodeText(bytes) : "");

  function download(): void {
    const blob = new Blob([toArrayBuffer(bytes)], {
      type: item.content_type || "application/octet-stream",
    });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = (item.name || "sharelock-item").replace(/[\\/:*?"<>|]/g, "_");
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function copy(
    value: string,
    target: "content" | "accompanying" = "content",
  ): Promise<void> {
    let state: "copied" | "error";
    try {
      await navigator.clipboard.writeText(value);
      state = "copied";
    } catch {
      state = "error";
    }
    if (target === "accompanying") {
      accompanyingCopyState = state;
    } else {
      copyState = state;
    }
  }
</script>

<article box-="square" shear-="top" is-="column" gap-="1">
  <div is-="row" wrap- align-="center between" gap-="1">
    <div is-="row" wrap- align-="center start" gap-="1">
      <span is-="badge" variant-="background0"
        ><Icon name="file" /> {item.name || `PAYLOAD ${String(index + 1).padStart(2, "0")}`}</span
      >
      <span is-="badge" variant-="blue">{item.content_type || "application/octet-stream"}</span>
      <span is-="badge" variant-="background2">{formatBytes(bytes.byteLength)}</span>
    </div>
    <div is-="row" wrap- align-="center end" gap-="1">
      <span aria-live="polite">
        {#if copyState === "copied"}<span is-="badge" variant-="green"
            ><Icon name="check" /> COPIED</span
          >{/if}
      </span>
      {#if isText}
        <button box-="round" type="button" onclick={() => void copy(text)}
          ><Icon name="copy" /> Copy</button
        >
      {/if}
      <button box-="round" variant-="blue" type="button" onclick={download}
        ><Icon name="download" /> Download</button
      >
    </div>
  </div>

  <div pad-="1" is-="column" gap-="1">
    {#if item.accompanying_text !== undefined}
      <section is-="column" gap-="1" aria-label="Accompanying text">
        <div is-="row" wrap- align-="center between" gap-="1">
          <strong>Accompanying text</strong>
          <div is-="row" wrap- align-="center end" gap-="1">
            <span aria-live="polite">
              {#if accompanyingCopyState === "copied"}
                <span is-="badge" variant-="green"><Icon name="check" /> COPIED</span>
              {/if}
            </span>
            <button
              box-="round"
              type="button"
              onclick={() => void copy(item.accompanying_text!, "accompanying")}
              ><Icon name="copy" /> Copy text</button
            >
          </div>
        </div>
        {#if accompanyingCopyState === "error"}
          <mark fg-="red" role="alert"
            >Clipboard access failed. Select the text and copy it manually.</mark
          >
        {/if}
        <pre>{item.accompanying_text}</pre>
      </section>
    {/if}
    {#if copyState === "error"}
      <mark fg-="red" role="alert"
        >Clipboard access failed. Select the content and copy it manually.</mark
      >
    {/if}
    {#if isText}
      <pre>{text}</pre>
    {:else}
      <mark fg-="foreground2">Binary payload. Use Download to save it locally.</mark>
    {/if}
  </div>
</article>

<style>
  pre {
    max-block-size: 24lh;
    overflow: auto;
    overflow-wrap: anywhere;
  }
</style>
