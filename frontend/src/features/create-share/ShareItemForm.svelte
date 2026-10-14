<script lang="ts">
  import { messages } from "../../shared/i18n/locale";
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
  let contentKind = $derived(
    item.file
      ? item.text.length > 0
        ? $messages.fileText
        : $messages.file
      : item.text.length > 0
        ? $messages.text
        : $messages.empty,
  );
  let contentTone = $derived(item.file ? "blue" : item.text.length > 0 ? "green" : "foreground2");

  function updateFile(event: Event): void {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0] ?? null;
    onChange({ ...item, file });
  }

  function updateText(event: Event): void {
    onChange({ ...item, text: (event.currentTarget as HTMLTextAreaElement).value });
  }

  function clearFile(): void {
    if (fileInput) {
      fileInput.value = "";
    }
    onChange({ ...item, file: null });
  }
</script>

<section box-="square" shear-="top" is-="column" gap-="1">
  <div is-="row" wrap- align-="center between" gap-="1">
    <div is-="row" wrap- align-="center start" gap-="1">
      <span is-="badge" variant-="background0"
        >{$messages.payload} {String(index + 1).padStart(2, "0")}</span
      >
      <span is-="badge" variant-={contentTone}>{contentKind}</span>
    </div>
    {#if removable}
      <button box-="round" variant-="red" type="button" onclick={onRemove} {disabled}>
        <Icon name="trash" />
        {$messages.remove}
      </button>
    {/if}
  </div>

  <div pad-="1" is-="column" gap-="1">
    <label is-="column" gap-="1">
      <span>{$messages.name} <mark fg-="foreground2">{$messages.optional}</mark></span>
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
      <span>{item.file ? $messages.accompanyingOptional : $messages.pasteText}</span>
      <textarea
        rows="7"
        placeholder={$messages.textPlaceholder}
        value={item.text}
        oninput={updateText}
        {disabled}></textarea>
    </label>

    <div is-="separator" variant-="foreground2">{$messages.fileOptional}</div>

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
        <button box-="round" type="button" onclick={clearFile} {disabled}
          >{$messages.clearFile}</button
        >
      </div>
    {/if}
    <label data-file-picker aria-disabled={disabled}>
      <input
        bind:this={fileInput}
        type="file"
        aria-label={item.file ? $messages.replaceFile : $messages.chooseFile}
        onchange={updateFile}
        {disabled}
      />
      <span is-="button" box-="round" variant-="blue"
        ><Icon name="file" /> {item.file ? $messages.replaceFile : $messages.chooseFile}</span
      >
      {#if !item.file}<mark fg-="foreground2">{$messages.noFile}</mark>{/if}
    </label>
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
