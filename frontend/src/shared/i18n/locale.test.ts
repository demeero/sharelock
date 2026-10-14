import { get } from "svelte/store";
import { describe, expect, it } from "vitest";

import { createLocale, detectLanguage } from "./locale";

describe("language selection", () => {
  it.each([
    ["uk", "uk"],
    ["uk-UA", "uk"],
    ["UK-ua", "uk"],
    ["ru", "uk"],
    ["ru-RU", "uk"],
    ["ru-UA", "uk"],
    ["en", "en"],
    ["en-US", "en"],
    ["de-DE", "en"],
    [undefined, "en"],
    ["", "en"],
  ])("resolves primary browser language %s to %s", (browserLanguage, expected) => {
    expect(detectLanguage(browserLanguage)).toBe(expected);
  });

  it("uses the primary language even when Ukrainian is a secondary preference", () => {
    const browserLanguages = ["en-US", "uk"];
    expect(get(createLocale(browserLanguages[0])).language).toBe("en");
  });

  it("remembers explicit choices and returning to Automatic across reloads", () => {
    const values = new Map<string, string>();
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => {
        values.set(key, value);
      },
    };
    const locale = createLocale("ru-RU", storage);
    expect(get(locale)).toEqual({ preference: "auto", language: "uk" });
    locale.setPreference("en");
    expect(get(createLocale("ru-RU", storage))).toEqual({ preference: "en", language: "en" });
    locale.setPreference("uk");
    expect(get(createLocale("en-US", storage))).toEqual({ preference: "uk", language: "uk" });
    locale.setPreference("auto");
    expect(get(createLocale("en-US", storage))).toEqual({ preference: "auto", language: "en" });
  });

  it("ignores an unsupported saved choice", () => {
    const locale = createLocale("uk-UA", { getItem: () => "fr", setItem: () => {} });
    expect(get(locale)).toEqual({ preference: "auto", language: "uk" });
  });

  it("keeps selection usable when storage is blocked", () => {
    const blocked = () => {
      throw new Error("Storage blocked");
    };
    const locale = createLocale("en", { getItem: blocked, setItem: blocked });
    expect(get(locale).language).toBe("en");
    locale.setPreference("uk");
    expect(get(locale)).toEqual({ preference: "uk", language: "uk" });
    locale.setPreference("auto");
    expect(get(locale).language).toBe("en");
  });
});
