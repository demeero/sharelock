import { derived, writable } from "svelte/store";

import { translations, type Language } from "./messages";

export type LanguagePreference = "auto" | Language;
const storageKey = "sharelock.language";
type PreferenceStorage = Pick<Storage, "getItem" | "setItem">;

export function detectLanguage(browserLanguage: string | undefined): Language {
  const base = browserLanguage?.toLowerCase().split("-")[0];
  return base === "uk" || base === "ru" ? "uk" : "en";
}

export function createLocale(browserLanguage: string | undefined, storage?: PreferenceStorage) {
  let preference: LanguagePreference = "auto";
  try {
    const saved = storage?.getItem(storageKey);
    if (saved === "auto" || saved === "en" || saved === "uk") preference = saved;
  } catch {
    // Storage may be blocked; language selection still works for this session.
  }
  const resolve = (value: LanguagePreference) => ({
    preference: value,
    language: value === "auto" ? detectLanguage(browserLanguage) : value,
  });
  const state = writable(resolve(preference));
  return {
    subscribe: state.subscribe,
    setPreference(value: LanguagePreference) {
      state.set(resolve(value));
      try {
        storage?.setItem(storageKey, value);
      } catch {
        // Keep the in-memory selection when persistence is unavailable.
      }
    },
  };
}

function browserStorage(): PreferenceStorage | undefined {
  try {
    return globalThis.localStorage;
  } catch {
    return undefined;
  }
}

export const locale = createLocale(globalThis.navigator?.language, browserStorage());
export const messages = derived(locale, ({ language }) => translations[language]);
