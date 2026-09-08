<script lang="ts">
  import type { Snippet } from "svelte";

  import Icon from "./Icon.svelte";

  interface Props {
    route: string;
    children: Snippet;
  }

  let { route, children }: Props = $props();
</script>

<main is-="column app-shell" gap-="1">
  <header box-="square" pad-="1">
    <div is-="row" wrap- align-="center between" gap-="1">
      <a data-brand href="/" aria-label="Sharelock home">
        <mark fg-="mauve"><Icon name="lock" /> SHARELOCK</mark>
      </a>
      <div is-="row" wrap- align-="center end" gap-="1">
        <span is-="badge" variant-="green" cap-="triangle">CLIENT-SIDE CRYPTO</span>
        <mark data-route fg-="foreground2">{route}</mark>
      </div>
    </div>
  </header>

  {@render children()}

  <footer is-="column" gap-="1">
    <div is-="separator" variant-="foreground2"></div>
    <mark fg-="foreground2"
      >AES-256-GCM · key stays in URL fragment · server stores ciphertext only</mark
    >
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

  footer {
    padding-inline: 1ch;
  }
</style>
