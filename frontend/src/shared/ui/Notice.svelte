<script lang="ts">
  import { messages } from "../i18n/locale";
  interface Props {
    status: string;
    message: string;
    tone?: "green" | "yellow" | "red" | "blue";
    recoveryHref?: string;
    recoveryLabel?: string;
  }

  let { status, message, tone = "red", recoveryHref, recoveryLabel }: Props = $props();
</script>

<aside box-="square" pad-="1" role={tone === "red" ? "alert" : "status"}>
  <div is-="row" wrap- align-="center between" gap-="1">
    <div is-="row" wrap- align-="center start" gap-="1">
      <span is-="badge" variant-={tone}>{status}</span>
      <span>{message}</span>
    </div>
    {#if recoveryHref}
      <a href={recoveryHref}>{recoveryLabel ?? $messages.createNew}</a>
    {/if}
  </div>
</aside>
