<script lang="ts">
  import { base64URLToBytes } from "../../shared/crypto/base64url";
  import { decodeText, toArrayBuffer, type PayloadBlock } from "../../shared/crypto/share-crypto";
  import { formatBytes } from "../../shared/format/bytes";
  import Icon from "../../shared/ui/Icon.svelte";

  interface Props {
    payloadBlock: PayloadBlock;
    index: number;
  }

  let { payloadBlock, index }: Props = $props();
  let copyState = $state<"idle" | "copied" | "error">("idle");
  let file = $derived(payloadBlock.file);
  let bytes = $derived(base64URLToBytes(file?.bytes_base64 ?? ""));
  let note = $derived(payloadBlock.text);
  let fileIsText = $derived(file?.content_type.startsWith("text/") === true);
  let fileText = $derived(fileIsText ? decodeText(bytes) : "");

  function download(): void {
    const contents =
      file === undefined ? new TextEncoder().encode(note ?? "") : toArrayBuffer(bytes);
    const blob = new Blob([contents], {
      type: file?.content_type || "text/plain;charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = (file?.name || "sharelock-note.txt").replace(/[\\/:*?"<>|]/g, "_");
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function copy(text: string): Promise<void> {
    try {
      await navigator.clipboard.writeText(text);
      copyState = "copied";
    } catch {
      copyState = "error";
    }
  }
</script>

<article box-="square" shear-="top" is-="column" gap-="1">
  {#if note !== undefined}
    <div pad-="1" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">TEXT NOTE</span>
        <div is-="row" wrap- align-="center end" gap-="1">
          <span aria-live="polite">
            {#if copyState === "copied"}<span is-="badge" variant-="green"
                ><Icon name="check" /> COPIED</span
              >{/if}
          </span>
          <button box-="round" type="button" onclick={() => void copy(note)}
            ><Icon name="copy" /> Copy</button
          >
          {#if !file}
            <button box-="round" variant-="blue" type="button" onclick={download}
              ><Icon name="download" /> Download</button
            >
          {/if}
        </div>
      </div>
      {#if copyState === "error"}
        <mark fg-="red" role="alert"
          >Clipboard access failed. Select the content and copy it manually.</mark
        >
      {/if}
      <pre>{note}</pre>
    </div>
  {/if}

  {#if file}
    <div pad-="1" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <div is-="row" wrap- align-="center start" gap-="1">
          <span is-="badge" variant-="background0"
            ><Icon name="file" />
            {file.name || `PAYLOAD ${String(index + 1).padStart(2, "0")}`}</span
          >
          <span is-="badge" variant-="blue">{file.content_type || "application/octet-stream"}</span>
          <span is-="badge" variant-="background2">{formatBytes(bytes.byteLength)}</span>
        </div>
        <div is-="row" wrap- align-="center end" gap-="1">
          {#if fileIsText && note === undefined}
            <span aria-live="polite">
              {#if copyState === "copied"}<span is-="badge" variant-="green"
                  ><Icon name="check" /> COPIED</span
                >{/if}
            </span>
            <button box-="round" type="button" onclick={() => void copy(fileText)}
              ><Icon name="copy" /> Copy</button
            >
          {/if}
          <button box-="round" variant-="blue" type="button" onclick={download}
            ><Icon name="download" /> Download</button
          >
        </div>
      </div>
      {#if copyState === "error" && note === undefined}
        <mark fg-="red" role="alert"
          >Clipboard access failed. Select the content and copy it manually.</mark
        >
      {/if}
      {#if fileIsText}
        <pre>{fileText}</pre>
      {:else}
        <mark fg-="foreground2">Binary payload. Use Download to save it locally.</mark>
      {/if}
    </div>
  {/if}
</article>

<style>
  pre {
    max-block-size: 24lh;
    overflow: auto;
    overflow-wrap: anywhere;
  }
</style>
