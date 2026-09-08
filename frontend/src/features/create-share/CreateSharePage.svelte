<script lang="ts">
  import { apiClient, apiError } from "../../shared/api/client";
  import { bytesToBase64URL } from "../../shared/crypto/base64url";
  import { encryptItems, type EncryptedItem } from "../../shared/crypto/share-crypto";
  import { formatBytes } from "../../shared/format/bytes";
  import AppShell from "../../shared/ui/AppShell.svelte";
  import CopyField from "../../shared/ui/CopyField.svelte";
  import Icon from "../../shared/ui/Icon.svelte";
  import Notice from "../../shared/ui/Notice.svelte";
  import ShareItemForm from "./ShareItemForm.svelte";
  import type { DraftItem } from "./types";

  const expiryLabels: Record<string, string> = {
    "3600": "1h",
    "86400": "24h",
    "604800": "7d",
    "2592000": "30d",
  };

  let nextID = 1;
  let items = $state<DraftItem[]>([newDraftItem()]);
  let expiresInSeconds = $state("86400");
  let burnAfterOpen = $state(false);
  let busy = $state(false);
  let error = $state("");
  let created = $state<{ revokeURL: string; shareURL: string } | null>(null);

  let payloadCount = $derived(items.filter((item) => item.file || item.text.length > 0).length);
  let totalBytes = $derived(
    items.reduce(
      (total, item) => total + (item.file?.size ?? new TextEncoder().encode(item.text).byteLength),
      0,
    ),
  );
  let summary = $derived(
    `${payloadCount} ${payloadCount === 1 ? "payload" : "payloads"} · ${formatBytes(totalBytes)} · expires in ${expiryLabels[expiresInSeconds]} · ${burnAfterOpen ? "burn after open" : "reusable"}`,
  );

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
    if (!created && !busy && (event.ctrlKey || event.metaKey) && event.key === "Enter") {
      event.preventDefault();
      void submit();
    }
  }

  async function submit(): Promise<void> {
    if (busy) {
      return;
    }

    error = "";
    busy = true;
    try {
      const encryptedItems = await collectItems();
      if (encryptedItems.length === 0) {
        throw new Error("Add at least one non-empty text value or file.");
      }
      const { envelope, key } = await encryptItems(encryptedItems);
      const { data, error: responseError } = await apiClient.POST("/api/v1/shares", {
        body: {
          envelope,
          expires_in_seconds: Number(expiresInSeconds),
          burn_after_open: burnAfterOpen,
        },
      });
      if (!data) {
        throw apiError(responseError, "Could not create the share.");
      }
      const baseURL = `${window.location.origin}/s/${data.id}`;
      created = {
        shareURL: `${baseURL}#k=${bytesToBase64URL(key)}`,
        revokeURL: `${baseURL}#revoke=${data.revoke_token}`,
      };
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
              <span is-="badge" variant-="background2">{expiryLabels[expiresInSeconds]}</span>
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
                <label
                  ><input
                    type="radio"
                    name="expires-after"
                    value="3600"
                    bind:group={expiresInSeconds}
                    disabled={busy}
                  /> 1 hour</label
                >
                <label
                  ><input
                    type="radio"
                    name="expires-after"
                    value="86400"
                    bind:group={expiresInSeconds}
                    disabled={busy}
                  /> 24 hours</label
                >
                <label
                  ><input
                    type="radio"
                    name="expires-after"
                    value="604800"
                    bind:group={expiresInSeconds}
                    disabled={busy}
                  /> 7 days</label
                >
                <label
                  ><input
                    type="radio"
                    name="expires-after"
                    value="2592000"
                    bind:group={expiresInSeconds}
                    disabled={busy}
                  /> 30 days</label
                >
              </div>
              <label
                ><input is-="switch" type="checkbox" bind:checked={burnAfterOpen} disabled={busy} /> Burn
                after first open</label
              >
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
      </div>
    </section>
  {/if}
</AppShell>
