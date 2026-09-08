<script lang="ts">
  import { apiClient, apiError } from "../../shared/api/client";
  import { decryptItems, type EncryptedItem } from "../../shared/crypto/share-crypto";
  import AppShell from "../../shared/ui/AppShell.svelte";
  import Icon from "../../shared/ui/Icon.svelte";
  import Notice from "../../shared/ui/Notice.svelte";
  import ShareContentItem from "./ShareContentItem.svelte";

  interface Props {
    shareID: string;
  }

  let { shareID }: Props = $props();

  const fragment = new URLSearchParams(window.location.hash.slice(1));
  const revokeToken = fragment.get("revoke");
  let items = $state<EncryptedItem[] | null>(null);
  let burned = $state(false);
  let busy = $state(false);
  let error = $state("");
  let errorStatus = $state("UNAVAILABLE");
  let revoked = $state(false);
  let revokeDialog = $state<HTMLDialogElement>();

  async function open(): Promise<void> {
    if (busy) {
      return;
    }

    error = "";
    busy = true;
    try {
      const key = fragment.get("k");
      if (!key) {
        errorStatus = "INVALID KEY";
        throw new Error("This URL does not contain a decryption key.");
      }

      const { data, error: responseError } = await apiClient.POST("/api/v1/shares/{id}/open", {
        params: { path: { id: shareID } },
      });
      if (!data) {
        errorStatus = "UNAVAILABLE";
        throw apiError(responseError, "This share is unavailable, expired, burned, or revoked.");
      }

      try {
        items = await decryptItems(data.envelope, key);
      } catch {
        errorStatus = "INVALID KEY";
        throw new Error("The decryption key is invalid or the encrypted payload is damaged.");
      }
      burned = data.burned;
    } catch (caught) {
      error = caught instanceof Error ? caught.message : "Could not decrypt this share.";
    } finally {
      busy = false;
    }
  }

  function requestRevoke(): void {
    error = "";
    revokeDialog?.showModal();
  }

  function cancelRevoke(): void {
    error = "";
    revokeDialog?.close();
  }

  async function revoke(): Promise<void> {
    if (!revokeToken || busy) {
      return;
    }

    error = "";
    busy = true;
    try {
      const { error: responseError } = await apiClient.DELETE("/api/v1/shares/{id}", {
        params: {
          path: { id: shareID },
          header: { "X-Revoke-Token": revokeToken },
        },
      });
      if (responseError) {
        throw apiError(responseError, "This share is already unavailable.");
      }
      revoked = true;
      revokeDialog?.close();
    } catch (caught) {
      error = caught instanceof Error ? caught.message : "Could not revoke this share.";
    } finally {
      busy = false;
    }
  }
</script>

<AppShell route={revokeToken ? "~/shares/revoke" : "~/shares/open"}>
  {#if revokeToken}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">REVOKE CAPABILITY</span>
        <span is-="badge" variant-="red"><Icon name="key" /> ADMIN</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        {#if revoked}
          <Notice
            status="REVOKED"
            tone="green"
            message="The encrypted blob was permanently removed. This share can no longer be opened."
            recoveryHref="/"
          />
        {:else}
          <div is-="column" gap-="1">
            <h1>Revoke encrypted share</h1>
            <p>
              This administrative capability permanently removes the encrypted blob from Sharelock.
            </p>
          </div>
          <Notice
            status="DANGER"
            message="Revocation cannot be undone. Confirm the exact share before continuing."
          />
          <div is-="row" wrap- align-="center end" gap-="1">
            <button box-="round" variant-="red" type="button" onclick={requestRevoke}
              ><Icon name="trash" /> Revoke share</button
            >
          </div>
        {/if}
      </div>
    </section>

    <dialog
      bind:this={revokeDialog}
      box-="double"
      size-="default"
      position-="center"
      container-="auto"
      oncancel={() => {
        error = "";
      }}
    >
      <div pad-="1" is-="column" gap-="2">
        <div is-="row" wrap- align-="center between" gap-="1">
          <h2>Confirm permanent revoke</h2>
          <span is-="badge" variant-="red">IRREVERSIBLE</span>
        </div>
        <p>
          The ciphertext will be deleted immediately. Recipients will lose access even if they still
          have the share URL.
        </p>
        {#if error}
          <Notice status="REVOKE FAILED" message={error} />
        {/if}
        <div is-="row" wrap- align-="center end" gap-="1">
          <button box-="round" type="button" onclick={cancelRevoke} disabled={busy}
            >Cancel <span is-="badge" variant-="background2">Esc</span></button
          >
          <button
            box-="round"
            variant-="red"
            type="button"
            onclick={() => void revoke()}
            disabled={busy}
          >
            {#if busy}<span is-="spinner" variant-="dots" aria-hidden="true"></span> Revoking…{:else}<Icon
                name="trash"
              /> Revoke permanently{/if}
          </button>
        </div>
      </div>
    </dialog>
  {:else if items}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">DECRYPTED</span>
        <span is-="badge" variant-="green"
          ><Icon name="check" /> {items.length} {items.length === 1 ? "ITEM" : "ITEMS"}</span
        >
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>Decrypted payloads</h1>
          <p>
            <mark fg-="foreground2"
              >Plaintext exists only in this browser tab. Save what you need, then close it.</mark
            >
          </p>
        </div>
        {#if burned}
          <Notice
            status="BURNED"
            tone="yellow"
            message="This share was deleted from the server after this open."
          />
        {/if}
        <div is-="column" gap-="2">
          {#each items as item, index (item.name + item.bytes_base64)}
            <ShareContentItem {item} {index} />
          {/each}
        </div>
      </div>
    </section>
  {:else}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">ENCRYPTED</span>
        <span is-="badge" variant-="mauve"><Icon name="lock" /> SEALED</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>Open encrypted share</h1>
          <p>
            The server returns ciphertext. The key in this URL decrypts it only inside this browser.
          </p>
        </div>
        {#if error}
          <Notice status={errorStatus} message={error} recoveryHref="/" />
        {/if}
        <div is-="row" wrap- align-="center end" gap-="1">
          <button
            box-="round"
            variant-="mauve"
            type="button"
            onclick={() => void open()}
            disabled={busy}
          >
            {#if busy}<span is-="spinner" variant-="dots" aria-hidden="true"></span> Decrypting…{:else}<Icon
                name="key"
              /> Open & decrypt{/if}
          </button>
        </div>
      </div>
    </section>
  {/if}
</AppShell>

<style>
  dialog::backdrop {
    background: color-mix(in srgb, var(--crust) 82%, transparent);
  }

  dialog {
    inline-size: calc(100% - 2ch);
  }
</style>
