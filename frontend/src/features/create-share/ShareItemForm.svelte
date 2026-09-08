<script lang="ts">
  import { formatBytes } from "../../shared/format/bytes";
  import Icon from "../../shared/ui/Icon.svelte";
  import type { DraftItem } from "./types";

  interface Props {
    item: DraftItem;
    index: number;
    removable: boolean;
    disabled?: boolean;
    onChange: (item: DraftItem) => void;
    onRemove: () => void;
  }

  let { item, index, removable, disabled = false, onChange, onRemove }: Props = $props();
  let fileInput = $state<HTMLInputElement>();
  let fileError = $state("");

  let contentKind = $derived(item.file ? "FILE" : item.text.length > 0 ? "TEXT" : "EMPTY");
  let contentTone = $derived(item.file ? "blue" : item.text.length > 0 ? "green" : "foreground2");
  let fileSelectionDisabled = $derived(disabled || item.text.length > 0);

  function updateFile(event: Event): void {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0] ?? null;
    if (file && item.text.length > 0) {
      fileError = "Clear the text before selecting a file.";
      input.value = "";
      return;
    }

    fileError = "";
    onChange({ ...item, file });
  }

  function updateText(event: Event): void {
    fileError = "";
    onChange({ ...item, text: (event.currentTarget as HTMLTextAreaElement).value });
  }

  function clearFile(): void {
    if (fileInput) {
      fileInput.value = "";
    }
    fileError = "";
    onChange({ ...item, file: null });
  }
</script>

<section box-="square" shear-="top" is-="column" gap-="1">
  <div is-="row" wrap- align-="center between" gap-="1">
    <div is-="row" wrap- align-="center start" gap-="1">
      <span is-="badge" variant-="background0">PAYLOAD {String(index + 1).padStart(2, "0")}</span>
      <span is-="badge" variant-={contentTone}>{contentKind}</span>
    </div>
    {#if removable}
      <button box-="round" variant-="red" type="button" onclick={onRemove} {disabled}>
        <Icon name="trash" /> Remove
      </button>
    {/if}
  </div>

  <div pad-="1" is-="column" gap-="1">
    <label is-="column" gap-="1">
      <span>Name <mark fg-="foreground2">(optional)</mark></span>
      <input
        maxlength="200"
        placeholder="production.env"
        value={item.name}
        oninput={(event) =>
          onChange({ ...item, name: (event.currentTarget as HTMLInputElement).value })}
        {disabled}
      />
    </label>

    <label is-="column" gap-="1">
      <span>Paste text</span>
      <textarea
        rows="7"
        placeholder="A password, config, certificate, or any other text"
        value={item.text}
        oninput={updateText}
        disabled={disabled || Boolean(item.file)}></textarea>
    </label>

    <div is-="separator" variant-="foreground2">OR</div>

    {#if item.file}
      <div data-file-details is-="row" wrap- align-="center between" gap-="1">
        <div is-="row" wrap- align-="center start" gap-="1">
          <Icon name="file" />
          <strong>{item.file.name}</strong>
          <span is-="badge" variant-="background2"
            >{item.file.type || "application/octet-stream"}</span
          >
          <span is-="badge" variant-="background2">{formatBytes(item.file.size)}</span>
        </div>
        <button box-="round" type="button" onclick={clearFile} {disabled}>Clear file</button>
      </div>
    {:else}
      <label data-file-picker aria-disabled={fileSelectionDisabled}>
        <input
          bind:this={fileInput}
          type="file"
          onchange={updateFile}
          disabled={fileSelectionDisabled}
        />
        <span is-="button" box-="round" variant-="blue"><Icon name="file" /> Choose file</span>
        <mark fg-="foreground2">
          {item.text.length > 0 ? "Clear pasted text to enable files" : "No file selected"}
        </mark>
      </label>
    {/if}

    {#if fileError}
      <mark fg-="red" role="alert">{fileError}</mark>
    {/if}
  </div>
</section>

<style>
  [data-file-picker] {
    position: relative;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1lh 1ch;
    padding: 1lh 1ch;
    cursor: pointer;
  }

  [data-file-picker][aria-disabled="true"] {
    cursor: not-allowed;
    opacity: 0.6;
  }

  [data-file-picker] input {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    clip-path: inset(50%);
  }

  [data-file-picker]:focus-within {
    outline: 2px solid var(--mauve);
    outline-offset: 2px;
  }

  [data-file-details] {
    min-inline-size: 0;
    padding: 1lh 1ch;
    background: var(--background1);
  }
</style>
