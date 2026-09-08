<script lang="ts">
  interface Props {
    status: string;
    message: string;
    tone?: "green" | "yellow" | "red" | "blue";
    recoveryHref?: string;
    recoveryLabel?: string;
  }

  let {
    status,
    message,
    tone = "red",
    recoveryHref,
    recoveryLabel = "Create a new share",
  }: Props = $props();
</script>

<aside box-="square" pad-="1" role={tone === "red" ? "alert" : "status"}>
  <div is-="row" wrap- align-="center between" gap-="1">
    <div is-="row" wrap- align-="center start" gap-="1">
      <span is-="badge" variant-={tone}>{status}</span>
      <span>{message}</span>
    </div>
    {#if recoveryHref}
      <a href={recoveryHref}>{recoveryLabel}</a>
    {/if}
  </div>
</aside>
