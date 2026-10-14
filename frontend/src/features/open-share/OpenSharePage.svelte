<script lang="ts">
  import { messages } from "../../shared/i18n/locale";
  import { errorMessage, LocalizedError } from "../../shared/i18n/errors";
  import { apiClient, apiError } from "../../shared/api/client";
  import {
    decryptItems,
    decryptPasswordProtectedItems,
    verifyPassword,
    type EncryptedItem,
  } from "../../shared/crypto/share-crypto";
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
  let viewsLeft = $state<number | null>(null);
  let busy = $state(false);
  let error = $state<unknown>(null);
  let errorStatus = $state<"unavailable" | "invalidKey" | "passwordInvalid">("unavailable");
  let passwordAccess = $state<string | null>(null);
  let password = $state("");
  let passwordVisible = $state(false);
  let revoked = $state(false);
  let revokeDialog = $state<HTMLDialogElement>();

  async function open(): Promise<void> {
    if (busy) {
      return;
    }

    error = null;
    busy = true;
    try {
      const key = fragment.get("k");
      if (!key) {
        errorStatus = "invalidKey";
        throw new LocalizedError((m) => m.missingKey);
      }

      const { data: accessData, error: responseError } = await apiClient.GET(
        "/api/v1/shares/{id}/access",
        { params: { path: { id: shareID } } },
      );
      if (!accessData) {
        errorStatus = "unavailable";
        throw apiError(responseError, new LocalizedError((m) => m.shareUnavailable));
      }
      if (accessData.access_envelope) {
        passwordAccess = accessData.access_envelope;
        return;
      }

      await claimAndDecrypt(key);
    } catch (caught) {
      error = caught;
    } finally {
      busy = false;
    }
  }

  async function openPasswordProtectedShare(): Promise<void> {
    if (busy || !passwordAccess) {
      return;
    }

    error = null;
    busy = true;
    try {
      const key = fragment.get("k");
      if (!key) {
        errorStatus = "invalidKey";
        throw new LocalizedError((m) => m.missingKey);
      }

      let cryptoKey: CryptoKey;
      try {
        cryptoKey = await verifyPassword(passwordAccess, key, password);
      } catch {
        errorStatus = "passwordInvalid";
        throw new LocalizedError((m) => m.incorrectPassword);
      }
      await claimAndDecrypt(key, cryptoKey);
      password = "";
    } catch (caught) {
      error = caught;
    } finally {
      busy = false;
    }
  }

  async function claimAndDecrypt(key: string, passwordKey?: CryptoKey): Promise<void> {
    const { data, error: responseError } = await apiClient.POST("/api/v1/shares/{id}/open", {
      params: { path: { id: shareID } },
    });
    if (!data) {
      errorStatus = "unavailable";
      throw apiError(responseError, new LocalizedError((m) => m.shareUnavailable));
    }
    try {
      items =
        passwordKey === undefined
          ? await decryptItems(data.envelope, key)
          : await decryptPasswordProtectedItems(data.envelope, passwordKey);
    } catch {
      errorStatus = "invalidKey";
      throw new LocalizedError((m) => m.damagedPayload);
    }
    burned = data.burned;
    viewsLeft = data.views_left;
  }

  function requestRevoke(): void {
    error = null;
    revokeDialog?.showModal();
  }

  function cancelRevoke(): void {
    error = null;
    revokeDialog?.close();
  }

  async function revoke(): Promise<void> {
    if (!revokeToken || busy) {
      return;
    }

    error = null;
    busy = true;
    try {
      const { error: responseError } = await apiClient.DELETE("/api/v1/shares/{id}", {
        params: {
          path: { id: shareID },
          header: { "X-Revoke-Token": revokeToken },
        },
      });
      if (responseError) {
        throw apiError(responseError, new LocalizedError((m) => m.alreadyUnavailable));
      }
      revoked = true;
      revokeDialog?.close();
    } catch (caught) {
      error = caught;
    } finally {
      busy = false;
    }
  }
</script>

<AppShell route={revokeToken ? "~/shares/revoke" : "~/shares/open"}>
  {#if revokeToken}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">{$messages.revokeCapability}</span>
        <span is-="badge" variant-="red"><Icon name="key" /> {$messages.admin}</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        {#if revoked}
          <Notice
            status={$messages.revoked}
            tone="green"
            message={$messages.revokedMessage}
            recoveryHref="/"
          />
        {:else}
          <div is-="column" gap-="1">
            <h1>{$messages.revokeTitle}</h1>
            <p>
              {$messages.revokeIntro}
            </p>
          </div>
          <Notice status={$messages.danger} message={$messages.revokeConfirmWarning} />
          <div is-="row" wrap- align-="center end" gap-="1">
            <button box-="round" variant-="red" type="button" onclick={requestRevoke}
              ><Icon name="trash" /> {$messages.revokeShare}</button
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
        error = null;
      }}
    >
      <div pad-="1" is-="column" gap-="2">
        <div is-="row" wrap- align-="center between" gap-="1">
          <h2>{$messages.confirmRevoke}</h2>
          <span is-="badge" variant-="red">{$messages.irreversible}</span>
        </div>
        <p>
          {$messages.revokeConsequences}
        </p>
        {#if error}
          <Notice
            status={$messages.revokeFailed}
            message={errorMessage(error, $messages, $messages.revokeError)}
          />
        {/if}
        <div is-="row" wrap- align-="center end" gap-="1">
          <button box-="round" type="button" onclick={cancelRevoke} disabled={busy}
            >{$messages.cancel} <span is-="badge" variant-="background2">Esc</span></button
          >
          <button
            box-="round"
            variant-="red"
            type="button"
            onclick={() => void revoke()}
            disabled={busy}
          >
            {#if busy}<span is-="spinner" variant-="dots" aria-hidden="true"></span>
              {$messages.revoking}{:else}<Icon name="trash" /> {$messages.revokePermanently}{/if}
          </button>
        </div>
      </div>
    </dialog>
  {:else if items}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">{$messages.decrypted}</span>
        <span is-="badge" variant-="green"
          ><Icon name="check" /> {$messages.itemCount(items.length)}</span
        >
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>{$messages.decryptedTitle}</h1>
          <p>
            <mark fg-="foreground2">{$messages.plaintextWarning}</mark>
          </p>
        </div>
        {#if burned}
          <Notice status={$messages.burned} tone="yellow" message={$messages.burnedMessage} />
        {:else if viewsLeft !== null}
          <Notice
            status={$messages.opensLeft}
            tone="yellow"
            message={$messages.remainingOpens(viewsLeft)}
          />
        {/if}
        <div is-="column" gap-="2">
          {#each items as item, index (index)}
            <ShareContentItem {item} {index} />
          {/each}
        </div>
      </div>
    </section>
  {:else}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">{$messages.encrypted}</span>
        <span is-="badge" variant-="mauve"><Icon name="lock" /> {$messages.sealed}</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>{$messages.openTitle}</h1>
          <p>
            {$messages.openIntro}
          </p>
        </div>
        {#if error}
          <Notice
            status={$messages[errorStatus]}
            message={errorMessage(error, $messages, $messages.decryptError)}
            recoveryHref="/"
          />
        {/if}
        {#if passwordAccess}
          <form
            is-="column"
            gap-="1"
            onsubmit={(event) => {
              event.preventDefault();
              void openPasswordProtectedShare();
            }}
          >
            <p>
              <mark fg-="foreground2">{$messages.passwordIntro}</mark>
            </p>
            <label is-="column" gap-="1">
              <span>{$messages.password}</span>
              <input
                type={passwordVisible ? "text" : "password"}
                autocomplete="current-password"
                bind:value={password}
                disabled={busy}
              />
            </label>
            <div is-="row" wrap- align-="center between" gap-="1">
              <button
                box-="round"
                type="button"
                onclick={() => (passwordVisible = !passwordVisible)}
                disabled={busy}
                >{passwordVisible ? $messages.hidePassword : $messages.showPassword}</button
              >
              <button box-="round" variant-="mauve" type="submit" disabled={busy}>
                {#if busy}<span is-="spinner" variant-="dots" aria-hidden="true"></span>
                  {$messages.verifying}{:else}<Icon name="key" /> {$messages.verifyOpen}{/if}
              </button>
            </div>
          </form>
        {:else}
          <div is-="row" wrap- align-="center end" gap-="1">
            <button
              box-="round"
              variant-="mauve"
              type="button"
              onclick={() => void open()}
              disabled={busy}
            >
              {#if busy}<span is-="spinner" variant-="dots" aria-hidden="true"></span>
                {$messages.preparing}{:else}<Icon name="key" /> {$messages.openDecrypt}{/if}
            </button>
          </div>
        {/if}
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
