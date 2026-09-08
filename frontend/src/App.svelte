<script lang="ts">
  import CreateSharePage from "./features/create-share/CreateSharePage.svelte";
  import OpenSharePage from "./features/open-share/OpenSharePage.svelte";

  function shareIDFromPath(pathname: string): string | null {
    const match = pathname.match(/^\/s\/([^/]+)$/);
    return match?.[1] ?? null;
  }

  const shareID = shareIDFromPath(window.location.pathname);
</script>

{#if shareID}
  <OpenSharePage {shareID} />
{:else}
  <CreateSharePage />
{/if}
