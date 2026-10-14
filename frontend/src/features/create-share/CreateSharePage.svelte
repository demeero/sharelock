<script lang="ts">
  import { messages } from "../../shared/i18n/locale";
  import { errorMessage, LocalizedError } from "../../shared/i18n/errors";
  import { onMount } from "svelte";

  import { apiClient, apiError } from "../../shared/api/client";
  import { bytesToBase64URL } from "../../shared/crypto/base64url";
  import { encryptedShareByteLength, encryptItems } from "../../shared/crypto/share-crypto";
  import { formatBytes } from "../../shared/format/bytes";
  import AppShell from "../../shared/ui/AppShell.svelte";
  import CopyField from "../../shared/ui/CopyField.svelte";
  import Icon from "../../shared/ui/Icon.svelte";
  import Notice from "../../shared/ui/Notice.svelte";
  import ShareItemForm from "./ShareItemForm.svelte";
  import { collectItems, draftItemsByteLength } from "./draft-items";
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
  let error = $state<unknown>(null);
  let created = $state<{ passwordProtected: boolean; revokeURL: string; shareURL: string } | null>(
    null,
  );
  let settings = $state<{
    max_encrypted_bytes: number;
    max_ttl_seconds: number;
    max_views: number;
  } | null>(null);
  let settingsLoading = $state(true);
  let settingsError = $state<unknown>(null);

  let payloadCount = $derived(items.filter((item) => item.file || item.text.length > 0).length);
  let expiryOptions = $derived(settings ? allowedExpiryOptions(settings.max_ttl_seconds) : []);
  let selectedExpiryLabel = $derived(formatDuration(Number(expiresInSeconds)));
  let totalBytes = $derived(draftItemsByteLength(items));
  // Clearing the number input leaves allowedOpens undefined, so the label falls
  // back instead of rendering "undefined opens" until a valid count is typed.
  let openPolicyLabel = $derived.by(() => {
    if (!limitOpens) {
      return $messages.unlimited;
    }
    if (!Number.isSafeInteger(allowedOpens)) {
      return $messages.limitNotSet;
    }

    return $messages.opens(allowedOpens!);
  });
  let summary = $derived(
    `${$messages.payloads(payloadCount)} · ${formatBytes(totalBytes)} · ${$messages.expiresIn(selectedExpiryLabel)} · ${openPolicyLabel}`,
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
    settingsError = null;
    try {
      const { data, error: responseError } = await apiClient.GET("/api/v1/shares/settings");
      if (!data) {
        settingsError = apiError(responseError, new LocalizedError((m) => m.settingsFailed));
        return;
      }

      const options = allowedExpiryOptions(data.max_ttl_seconds);
      if (!Number.isSafeInteger(data.max_encrypted_bytes) || data.max_encrypted_bytes <= 0) {
        settingsError = new LocalizedError((m) => m.invalidSize);
        return;
      }
      if (options.length === 0) {
        settingsError = new LocalizedError((m) => m.invalidLifetime);
        return;
      }
      if (!Number.isSafeInteger(data.max_views) || data.max_views < 1) {
        settingsError = new LocalizedError((m) => m.invalidOpens);
        return;
      }

      settings = data;
      expiresInSeconds = String(defaultExpirySeconds(options));
    } catch (caught) {
      settingsError = caught;
    } finally {
      settingsLoading = false;
    }
  }

  async function submit(): Promise<void> {
    if (busy || !settings) {
      return;
    }

    error = null;
    busy = true;
    try {
      const encryptedItems = await collectItems(items);
      if (encryptedItems.length === 0) {
        throw new LocalizedError((m) => m.addContent);
      }
      if (limitOpens && !isAllowedOpenCount(allowedOpens, settings.max_views)) {
        throw new LocalizedError((m) => m.chooseOpens(settings!.max_views));
      }

      if (passwordProtected && password.length === 0) {
        throw new LocalizedError((m) => m.enterPassword);
      }
      if (passwordProtected && password !== passwordConfirmation) {
        throw new LocalizedError((m) => m.passwordMismatch);
      }

      const sharePassword = passwordProtected ? password : undefined;
      const expectedEnvelopeBytes = encryptedShareByteLength(encryptedItems, sharePassword);
      if (expectedEnvelopeBytes > settings.max_encrypted_bytes) {
        throw new LocalizedError((m) =>
          m.tooLargeExpected(
            formatBytes(expectedEnvelopeBytes),
            formatBytes(settings!.max_encrypted_bytes),
          ),
        );
      }

      const encryptedShare = await encryptItems(encryptedItems, sharePassword);
      const encryptedBytes =
        new TextEncoder().encode(encryptedShare.envelope).byteLength +
        new TextEncoder().encode(encryptedShare.accessEnvelope ?? "").byteLength;
      if (encryptedBytes > settings.max_encrypted_bytes) {
        throw new LocalizedError((m) =>
          m.tooLarge(formatBytes(encryptedBytes), formatBytes(settings!.max_encrypted_bytes)),
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
        throw apiError(responseError, new LocalizedError((m) => m.createError));
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
      error = caught;
    } finally {
      busy = false;
    }
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
    return $messages.duration(seconds);
  }
</script>

<svelte:window onkeydown={handleShortcut} />

<AppShell route={created ? "~/shares/ready" : "~/shares/new"}>
  {#if created}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">{$messages.shareReady}</span>
        <span is-="badge" variant-="green"><Icon name="check" /> {$messages.ready}</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>{$messages.createTitle}</h1>
          <p>{$messages.stored}</p>
        </div>

        <CopyField label={$messages.shareURL} value={created.shareURL} />

        {#if created.passwordProtected}
          <Notice
            status={$messages.passwordRequired}
            tone="yellow"
            message={$messages.passwordDelivery}
          />
        {/if}

        <section box-="square" shear-="top" is-="column" gap-="1">
          <div is-="row" wrap- align-="center between" gap-="1">
            <span is-="badge" variant-="background0">{$messages.revokeCapability}</span>
            <span is-="badge" variant-="red"><Icon name="key" /> {$messages.admin}</span>
          </div>
          <div pad-="1" is-="column" gap-="1">
            <p>{$messages.revokeWarning}</p>
            <CopyField label={$messages.revokeURL} value={created.revokeURL} tone="red" />
          </div>
        </section>

        <Notice status={$messages.oneTime} tone="yellow" message={$messages.linksWarning} />

        <div is-="row" wrap- align-="center end" gap-="1">
          <a href="/">{$messages.createAnother}</a>
        </div>
      </div>
    </section>
  {:else}
    <section box-="square" shear-="top" is-="column" gap-="1">
      <div is-="row" wrap- align-="center between" gap-="1">
        <span is-="badge" variant-="background0">{$messages.encrypted}</span>
        <span is-="badge" variant-="mauve"><Icon name="lock" /> AES-256-GCM</span>
      </div>

      <div pad-="2 1" is-="column" gap-="2">
        <div is-="column" gap-="1">
          <h1>{$messages.createTitle}</h1>
          <p>
            <mark fg-="foreground2">{$messages.createIntro}</mark>
          </p>
          <details is-="accordion">
            <summary>{$messages.privacy}</summary>
            <p>
              {$messages.privacyBefore} <code>#</code>{$messages.privacyAfter}
            </p>
          </details>
        </div>

        {#if settingsLoading}
          <div is-="row" align-="center start" gap-="1" aria-live="polite">
            <span is-="spinner" variant-="dots" aria-hidden="true"></span>
            <span>{$messages.loadingSettings}</span>
          </div>
        {:else if settingsError}
          <div is-="column" gap-="1">
            <Notice
              status={$messages.settingsUnavailable}
              message={errorMessage(settingsError, $messages, $messages.settingsFailed)}
            />
            <div is-="row" wrap- align-="center start" gap-="1">
              <button box-="round" type="button" onclick={() => void loadSettings()}
                >{$messages.retry}</button
              >
            </div>
          </div>
        {:else if settings}
          <form
            novalidate
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
                disabled={busy}>{$messages.addPayload}</button
              >
            </div>

            <section box-="square" shear-="top" is-="column" gap-="1">
              <div is-="row" wrap- align-="center between" gap-="1">
                <span is-="badge" variant-="background0">{$messages.deliveryPolicy}</span>
                <div is-="row" wrap- align-="center end" gap-="1">
                  <span is-="badge" variant-="background2">{selectedExpiryLabel}</span>
                  <span is-="badge" variant-="background2">{openPolicyLabel}</span>
                </div>
              </div>
              <div pad-="1" is-="column" gap-="1">
                <p>{$messages.expiresAfter}</p>
                <div
                  is-="row"
                  wrap-
                  align-="center start"
                  gap-="1"
                  role="radiogroup"
                  aria-label={$messages.expiresAfter}
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
                  ><input is-="switch" type="checkbox" bind:checked={limitOpens} disabled={busy} />
                  {$messages.limitOpens}</label
                >
                {#if limitOpens}
                  <label is-="column" gap-="1">
                    <span
                      >{$messages.opensAllowed}
                      <mark fg-="foreground2">(1-{settings.max_views})</mark></span
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
                    <mark fg-="foreground2">{$messages.lastOpenDeletion}</mark>
                  </p>
                {/if}

                <label>
                  <input
                    is-="switch"
                    type="checkbox"
                    bind:checked={passwordProtected}
                    disabled={busy}
                  />
                  {$messages.protectPassword}
                </label>
                {#if passwordProtected}
                  <div is-="column" gap-="1">
                    <label is-="column" gap-="1">
                      <span>{$messages.password}</span>
                      <input
                        type={passwordsVisible ? "text" : "password"}
                        autocomplete="new-password"
                        bind:value={password}
                        disabled={busy}
                      />
                    </label>
                    <label is-="column" gap-="1">
                      <span>{$messages.confirmPassword}</span>
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
                        >{passwordsVisible
                          ? $messages.hidePasswords
                          : $messages.showPasswords}</button
                      >
                    </div>
                    <p>
                      <mark fg-="foreground2">{$messages.passwordPrivacy}</mark>
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
                status={$messages.createFailed}
                message={errorMessage(error, $messages, $messages.createError)}
                recoveryHref="/"
                recoveryLabel={$messages.resetForm}
              />
            {/if}

            <div is-="row" wrap- align-="center between" gap-="1">
              <span is-="badge" variant-="background2">Ctrl/Cmd + Enter</span>
              <button box-="round" variant-="mauve" type="submit" disabled={busy}>
                {#if busy}<span is-="spinner" variant-="dots" aria-hidden="true"></span>
                  {$messages.encrypting}{:else}<Icon name="lock" /> {$messages.encryptCreate}{/if}
              </button>
            </div>
          </form>
        {/if}
      </div>
    </section>
  {/if}
</AppShell>
