<script lang="ts">
  import { locale, messages } from "../i18n/locale";
  import type { Language } from "../i18n/messages";
  import type { Snippet } from "svelte";

  import Icon from "./Icon.svelte";

  interface Props {
    route: string;
    children: Snippet;
  }

  let { route, children }: Props = $props();
  const languageOptions = [
    { value: "uk", label: "UA", name: "Українська" },
    { value: "en", label: "EN", name: "English" },
  ] satisfies { value: Language; label: string; name: string }[];

  function selectLanguage(language: Language): void {
    // Clicking an already-checked, detected language must also remember it explicitly.
    if ($locale.preference !== language) locale.setPreference(language);
  }
</script>

<main is-="column app-shell" gap-="1">
  <header box-="square" pad-="1">
    <div is-="row" wrap- align-="center between" gap-="1">
      <a data-brand href="/" aria-label={$messages.home}>
        <mark fg-="mauve"><Icon name="lock" /> SHARELOCK</mark>
      </a>
      <div is-="row" wrap- align-="center end" gap-="1">
        <span is-="badge" variant-="green" cap-="triangle">{$messages.clientCrypto}</span>
        <div data-language role="radiogroup" aria-label={$messages.language}>
          {#each languageOptions as option (option.value)}
            <label
              is-="button"
              size-="small"
              variant-={$locale.language === option.value ? "mauve" : "background0"}
              title={option.name}
            >
              <input
                type="radio"
                name="interface-language"
                value={option.value}
                aria-label={option.name}
                checked={$locale.language === option.value}
                onclick={() => selectLanguage(option.value)}
                onchange={() => selectLanguage(option.value)}
              />
              <span lang={option.value}>{option.label}</span>
            </label>
          {/each}
        </div>
        <mark data-route fg-="foreground2">{route}</mark>
      </div>
    </div>
  </header>

  {@render children()}

  <footer is-="column" gap-="1">
    <div is-="separator" variant-="foreground2"></div>
    <mark fg-="foreground2">{$messages.footer}</mark>
  </footer>
</main>

<style>
  [data-brand] {
    color: inherit;
    font-weight: var(--font-weight-bold);
    letter-spacing: 0.08ch;
    text-decoration: none;
    white-space: nowrap;
  }

  [data-route] {
    overflow-wrap: anywhere;
  }

  [data-language] {
    display: inline-flex;
    align-items: center;
    flex: none;
    padding: 0.15lh 0.3ch;
    gap: 0.3ch;
    border: 1px solid var(--background3);
    background: var(--background0);
  }

  [data-language] label {
    position: relative;
    min-inline-size: 4ch;
    block-size: 1.5lh;
    cursor: pointer;
    white-space: nowrap;
  }

  [data-language] label:hover {
    color: var(--mauve);
    background: var(--background1);
  }

  [data-language] label:has(input:checked) {
    color: var(--background0);
    background: var(--mauve);
  }

  [data-language] label:has(input:focus-visible) {
    outline: 2px solid var(--mauve);
    outline-offset: 2px;
  }

  [data-language] input {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    clip-path: inset(50%);
  }

  footer {
    padding-inline: 1ch;
  }
</style>
