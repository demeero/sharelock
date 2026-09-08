<script lang="ts">
  import Icon from "./Icon.svelte";

  interface Props {
    label: string;
    value: string;
    tone?: "blue" | "red";
  }

  let { label, value, tone = "blue" }: Props = $props();
  let state = $state<"idle" | "copied" | "error">("idle");

  async function copy(): Promise<void> {
    try {
      await navigator.clipboard.writeText(value);
      state = "copied";
    } catch {
      state = "error";
    }
  }
</script>

<div is-="column" gap-="1">
  <div is-="row" wrap- align-="center between" gap-="1">
    <label for={`copy-field-${label}`}>{label}</label>
    <span aria-live="polite">
      {#if state === "copied"}
        <span is-="badge" variant-="green"><Icon name="check" /> COPIED</span>
      {:else if state === "error"}
        <span is-="badge" variant-="red">COPY FAILED</span>
      {/if}
    </span>
  </div>
  <div data-copy-row is-="row" wrap- align-="stretch start" gap-="1">
    <input id={`copy-field-${label}`} readonly {value} />
    <button box-="round" variant-={tone} type="button" onclick={() => void copy()}>
      <Icon name="copy" /> Copy
    </button>
  </div>
  {#if state === "error"}
    <mark fg-="red" role="alert"
      >Clipboard access failed. Select the value and copy it manually.</mark
    >
  {/if}
</div>

<style>
  [data-copy-row] input {
    min-inline-size: min(100%, 28ch);
    flex: 1 1 28ch;
  }

  [data-copy-row] button {
    flex: none;
  }
</style>
