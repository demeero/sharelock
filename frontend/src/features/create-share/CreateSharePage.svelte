<script lang="ts">
  import { onMount } from "svelte";

  import { apiClient, apiError } from "../../shared/api/client";
  import { bytesToBase64URL } from "../../shared/crypto/base64url";
  import {
    encryptedShareByteLength,
    encryptItems,
    type EncryptedItem,
  } from "../../shared/crypto/share-crypto";
  import { formatBytes } from "../../shared/format/bytes";
  import AppShell from "../../shared/ui/AppShell.svelte";
  import CopyField from "../../shared/ui/CopyField.svelte";
  import Icon from "../../shared/ui/Icon.svelte";
  import Notice from "../../shared/ui/Notice.svelte";
  import ShareItemForm from "./ShareItemForm.svelte";
  import type { DraftItem } from "./types";

  const preferredExpirySeconds = [3600, 86400, 604800, 2592000];

  type ExpiryOption = {
    label: string;
    seconds: number;
  };

  let nextID = 1;
  let items = $state<DraftItem[]>([newDraftItem()]);
  let expiresInSeconds = $state("");
  let limitOpens = $state(false);
  let allowedOpens = $state<number | undefined>(1);
  let passwordProtected = $state(false);
  let password = $state("");
  let passwordConfirmation = $state("");
  let passwordsVisible = $state(false);
  let busy = $state(false);
  let error = $state("");
  let created = $state<{ passwordProtected: boolean; revokeURL: string; shareURL: string } | null>(
    null,
  );
  let settings = $state<{
    max_encrypted_bytes: number;
    max_ttl_seconds: number;
    max_views: number;
  } | null>(null);
  let settingsLoading = $state(true);
  let settingsError = $state("");

  let payloadCount = $derived(items.filter((item) => item.file || item.text.length > 0).length);
  let expiryOptions = $derived(settings ? allowedExpiryOptions(settings.max_ttl_seconds) : []);
  let selectedExpiryLabel = $derived(formatDuration(Number(expiresInSeconds)));
  let totalBytes = $derived(
    items.reduce(
      (total, item) => total + (item.file?.size ?? new TextEncoder().encode(item.text).byteLength),
      0,
    ),
  );
  // Clearing the number input leaves allowedOpens undefined, so the label falls
  // back instead of rendering "undefined opens" until a valid count is typed.
  let openPolicyLabel = $derived.by(() => {
    if (!limitOpens) {
      return "unlimited opens";
    }
    if (!Number.isSafeInteger(allowedOpens)) {
      return "open limit not set";
    }

    return `${allowedOpens} ${allowedOpens === 1 ? "open" : "opens"}`;
  });
  let summary = $derived(
    `${payloadCount} ${payloadCount === 1 ? "payload" : "payloads"} · ${formatBytes(totalBytes)} · expires in ${selectedExpiryLabel} · ${openPolicyLabel}`,
  );

  onMount(() => {
    void loadSettings();
  });

  function newDraftItem(): DraftItem {
    return { id: nextID++, name: "", text: "", file: null };
  }

  function updateItem(updated: DraftItem): void {
    items = items.map((item) => (item.id === updated.id ? updated : item));
  }

  function removeItem(id: number): void {
    if (!busy && items.length > 1) {
      items = items.filter((item) => item.id !== id);
    }
  }

  function handleShortcut(event: KeyboardEvent): void {
    if (
      !created &&
      settings &&
      !busy &&
      (event.ctrlKey || event.metaKey) &&
      event.key === "Enter"
    ) {
      event.preventDefault();
      void submit();
    }
  }

  async function loadSettings(): Promise<void> {
    settingsLoading = true;
    settingsError = "";
    try {
      const { data, error: responseError } = await apiClient.GET("/api/v1/shares/settings");
      if (!data) {
        settingsError = apiError(responseError, "Could not load share settings.").message;
        return;
      }

      const options = allowedExpiryOptions(data.max_ttl_seconds);
      if (!Number.isSafeInteger(data.max_encrypted_bytes) || data.max_encrypted_bytes <= 0) {
        settingsError = "The server returned an invalid encrypted share size limit.";
        return;
      }
      if (options.length === 0) {
        settingsError = "The server returned an invalid maximum share lifetime.";
        return;
      }
      if (!Number.isSafeInteger(data.max_views) || data.max_views < 1) {
        settingsError = "The server returned an invalid maximum number of opens.";
        return;
      }

      settings = data;
      expiresInSeconds = String(defaultExpirySeconds(options));
    } catch (caught) {
      settingsError = caught instanceof Error ? caught.message : "Could not load share settings.";
    } finally {
      settingsLoading = false;
    }
  }

  async function submit(): Promise<void> {
    if (busy || !settings) {
      return;
    }

    error = "";
    busy = true;
    try {
      const encryptedItems = await collectItems();
      if (encryptedItems.length === 0) {
        throw new Error("Add at least one non-empty text value or file.");
      }
      if (limitOpens && !isAllowedOpenCount(allowedOpens, settings.max_views)) {
        throw new Error(`Choose between 1 and ${settings.max_views} opens.`);
      }

      if (passwordProtected && password.length === 0) {
        throw new Error("Enter a password to protect this share.");
      }
      if (passwordProtected && password !== passwordConfirmation) {
        throw new Error("The password confirmation does not match.");
      }

      const sharePassword = passwordProtected ? password : undefined;
      const expectedEnvelopeBytes = encryptedShareByteLength(encryptedItems, sharePassword);
      if (expectedEnvelopeBytes > settings.max_encrypted_bytes) {
        throw new Error(
          `The encrypted share would be ${formatBytes(expectedEnvelopeBytes)}. The server limit is ${formatBytes(settings.max_encrypted_bytes)}.`,
        );
      }

      const encryptedShare = await encryptItems(encryptedItems, sharePassword);
      const encryptedBytes =
        new TextEncoder().encode(encryptedShare.envelope).byteLength +
        new TextEncoder().encode(encryptedShare.accessEnvelope ?? "").byteLength;
      if (encryptedBytes > settings.max_encrypted_bytes) {
        throw new Error(
          `The encrypted share is ${formatBytes(encryptedBytes)}. The server limit is ${formatBytes(settings.max_encrypted_bytes)}.`,
        );
      }
      const { data, error: responseError } = await apiClient.POST("/api/v1/shares", {
        body: {
          envelope: encryptedShare.envelope,
          access_envelope: encryptedShare.accessEnvelope,
          expires_in_seconds: Number(expiresInSeconds),
          views: limitOpens ? allowedOpens : undefined,
        },
      });
      if (!data) {
        throw apiError(responseError, "Could not create the share.");
      }
      const baseURL = `${window.location.origin}/s/${data.id}`;
      created = {
        passwordProtected,
        shareURL: `${baseURL}#k=${bytesToBase64URL(encryptedShare.key)}`,
        revokeURL: `${baseURL}#revoke=${data.revoke_token}`,
      };
      password = "";
      passwordConfirmation = "";
    } catch (caught) {
      error = caught instanceof Error ? caught.message : "Could not create the share.";
    } finally {
      busy = false;
    }
  }

  async function collectItems(): Promise<EncryptedItem[]> {
    const result: EncryptedItem[] = [];
    for (const [index, item] of items.entries()) {
      if (item.file && item.text.length > 0) {
        throw new Error(
          `Payload ${index + 1} contains both text and a file. Clear one before continuing.`,
        );
      }
      if (!item.file && item.text.length === 0) {
        continue;
      }

      const bytes = item.file
        ? new Uint8Array(await item.file.arrayBuffer())
        : new TextEncoder().encode(item.text);
      result.push({
        name: item.name.trim() || item.file?.name || `item-${index + 1}.txt`,
        content_type: item.file?.type || "text/plain;charset=utf-8",
        bytes_base64: bytesToBase64URL(bytes),
      });
    }

    return result;
  }

  function allowedExpiryOptions(maxTTLSeconds: number): ExpiryOption[] {
    if (!Number.isSafeInteger(maxTTLSeconds) || maxTTLSeconds <= 0) {
      return [];
    }

    const values = preferredExpirySeconds.filter((seconds) => seconds <= maxTTLSeconds);
    if (!values.includes(maxTTLSeconds)) {
      values.push(maxTTLSeconds);
    }

    return values
      .sort((left, right) => left - right)
      .map((seconds) => ({
        seconds,
        label: formatDuration(seconds),
      }));
  }

  function isAllowedOpenCount(count: number | undefined, maxViews: number): count is number {
    return count !== undefined && Number.isSafeInteger(count) && count >= 1 && count <= maxViews;
  }

  function defaultExpirySeconds(options: ExpiryOption[]): number {
    return options.find((option) => option.seconds === 86400)?.seconds ?? options.at(-1)!.seconds;
  }

  function formatDuration(seconds: number): string {
    if (seconds % 86400 === 0) {
      return `${seconds / 86400} day${seconds === 86400 ? "" : "s"}`;
    }
    if (seconds % 3600 === 0) {
      return `${seconds / 3600} hour${seconds === 3600 ? "" : "s"}`;
    }
    if (seconds % 60 === 0) {
      return `${seconds / 60} minute${seconds === 60 ? "" : "s"}`;
    }

    return `${seconds} second${seconds === 1 ? "" : "s"}`;
  }
</script>

<svelte:window onkeydown={handleShortcut} />

<AppShell route={created ? "~/shares/ready" : "~/shares/new"}>
  {#if created}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">SHARE READY</span>
        <span is-="badge" variant-="green"><Icon name="check" /> READY</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>Create encrypted share</h1>
          <p>The ciphertext is stored. The decryption key exists only in the complete share URL.</p>
        </div>

        <CopyField label="Share URL" value={created.shareURL} />

        {#if created.passwordProtected}
          <Notice
            status="PASSWORD REQUIRED"
            tone="yellow"
            message="The recipient must enter the password you chose. Send it through a separate trusted channel; it is not part of this URL."
          />
        {/if}

        <section box-="square" shear-="top" is-="column" gap-="1">
          <div is-="row" wrap- align-="center between" gap-="1">
            <span is-="badge" variant-="background0">REVOKE CAPABILITY</span>
            <span is-="badge" variant-="red"><Icon name="key" /> ADMIN</span>
          </div>
          <div pad-="1" is-="column" gap-="1">
            <p>Anyone holding this URL can permanently revoke the share. Store it separately.</p>
            <CopyField label="Revoke URL" value={created.revokeURL} tone="red" />
          </div>
        </section>

        <Notice
          status="ONE-TIME OUTPUT"
          tone="yellow"
          message="Neither URL can be recovered later. Send the share URL only through a trusted channel."
        />

        <div is-="row" wrap- align-="center end" gap-="1">
          <a href="/">Create another share</a>
        </div>
      </div>
    </section>
  {:else}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">ENCRYPTED</span>
        <span is-="badge" variant-="mauve"><Icon name="lock" /> AES-256-GCM</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>Create encrypted share</h1>
          <p>
            <mark fg-="foreground2"
              >Paste text or attach a file. Encryption happens before data leaves this browser.</mark
            >
          </p>
          <details is-="accordion">
            <summary>Privacy model</summary>
            <p>
              The share URL includes a decryption key after <code>#</code>. That fragment never
              reaches Sharelock; send the complete URL only through a trusted channel.
            </p>
          </details>
        </div>

        {#if settingsLoading}
          <div is-="row" align-="center start" gap-="1" aria-live="polite">
            <span is-="spinner" variant-="dots" aria-hidden="true"></span>
            <span>Loading server settings…</span>
          </div>
        {:else if settingsError}
          <div is-="column" gap-="1">
            <Notice status="SETTINGS UNAVAILABLE" message={settingsError} />
            <div is-="row" wrap- align-="center start" gap-="1">
              <button box-="round" type="button" onclick={() => void loadSettings()}>Retry</button>
            </div>
          </div>
        {:else if settings}
          <form
            is-="column"
            gap-="2"
            onsubmit={(event) => {
              event.preventDefault();
              void submit();
            }}
          >
            <div is-="column" gap-="2">
              {#each items as item, index (item.id)}
                <ShareItemForm
                  {item}
                  {index}
                  removable={items.length > 1}
                  disabled={busy}
                  onChange={updateItem}
                  onRemove={() => removeItem(item.id)}
                />
              {/each}
            </div>

            <div is-="row" wrap- align-="center start" gap-="1">
              <button
                box-="round"
                type="button"
                onclick={() => (items = [...items, newDraftItem()])}
                disabled={busy}>+ Add payload</button
              >
            </div>

            <section box-="square" shear-="top" is-="column" gap-="1">
              <div is-="row" wrap- align-="center between" gap-="1">
                <span is-="badge" variant-="background0">DELIVERY POLICY</span>
                <div is-="row" wrap- align-="center end" gap-="1">
                  <span is-="badge" variant-="background2">{selectedExpiryLabel}</span>
                  <span is-="badge" variant-="background2">{openPolicyLabel}</span>
                </div>
              </div>
              <div pad-="1" is-="column" gap-="1">
                <p>Expires after</p>
                <div
                  is-="row"
                  wrap-
                  align-="center start"
                  gap-="1"
                  role="radiogroup"
                  aria-label="expires after"
                >
                  {#each expiryOptions as option (option.seconds)}
                    <label
                      ><input
                        type="radio"
                        name="expires-after"
                        value={String(option.seconds)}
                        bind:group={expiresInSeconds}
                        disabled={busy}
                      />
                      {option.label}</label
                    >
                  {/each}
                </div>
                <label
                  ><input is-="switch" type="checkbox" bind:checked={limitOpens} disabled={busy} /> Limit
                  number of opens</label
                >
                {#if limitOpens}
                  <label is-="column" gap-="1">
                    <span
                      >Opens allowed <mark fg-="foreground2">(1-{settings.max_views})</mark></span
                    >
                    <input
                      type="number"
                      min="1"
                      max={settings.max_views}
                      step="1"
                      bind:value={allowedOpens}
                      disabled={busy}
                    />
                  </label>
                  <p>
                    <mark fg-="foreground2"
                      >The share is deleted from the server on its last allowed open.</mark
                    >
                  </p>
                {/if}

                <label>
                  <input
                    is-="switch"
                    type="checkbox"
                    bind:checked={passwordProtected}
                    disabled={busy}
                  />
                  Protect with a separate password
                </label>
                {#if passwordProtected}
                  <div is-="column" gap-="1">
                    <label is-="column" gap-="1">
                      <span>Password</span>
                      <input
                        type={passwordsVisible ? "text" : "password"}
                        autocomplete="new-password"
                        bind:value={password}
                        disabled={busy}
                      />
                    </label>
                    <label is-="column" gap-="1">
                      <span>Confirm password</span>
                      <input
                        type={passwordsVisible ? "text" : "password"}
                        autocomplete="new-password"
                        bind:value={passwordConfirmation}
                        disabled={busy}
                      />
                    </label>
                    <div is-="row" wrap- align-="center start" gap-="1">
                      <button
                        box-="round"
                        type="button"
                        onclick={() => (passwordsVisible = !passwordsVisible)}
                        disabled={busy}
                        >{passwordsVisible ? "Hide passwords" : "Show passwords"}</button
                      >
                    </div>
                    <p>
                      <mark fg-="foreground2"
                        >The password is used only in this browser and is never included in the
                        share URL or upload.</mark
                      >
                    </p>
                  </div>
                {/if}
              </div>
            </section>

            <div aria-live="polite">
              <mark fg-="foreground2">{summary}</mark>
            </div>

            {#if error}
              <Notice
                status="CREATE FAILED"
                message={error}
                recoveryHref="/"
                recoveryLabel="Reset form"
              />
            {/if}

            <div is-="row" wrap- align-="center between" gap-="1">
              <span is-="badge" variant-="background2">Ctrl/Cmd + Enter</span>
              <button box-="round" variant-="mauve" type="submit" disabled={busy}>
                {#if busy}<span is-="spinner" variant-="dots" aria-hidden="true"></span> Encrypting…{:else}<Icon
                    name="lock"
                  /> Encrypt & create{/if}
              </button>
            </div>
          </form>
        {/if}
      </div>
    </section>
  {/if}
</AppShell>
