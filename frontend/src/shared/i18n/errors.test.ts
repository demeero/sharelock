import { describe, expect, it } from "vitest";

import { apiError } from "../api/client";
import { errorMessage, LocalizedError } from "./errors";
import { en, uk } from "./messages";

describe("interface errors", () => {
  it("preserves server-provided text verbatim in both languages", () => {
    const detail = "invalid data: expiration must be between 1 and 604800 seconds";
    const error = apiError(
      { type: "about:blank", detail },
      new LocalizedError((m) => m.createError),
    );
    expect(errorMessage(error, en, en.createError)).toBe(detail);
    expect(errorMessage(error, uk, uk.createError)).toBe(detail);
  });

  it("localizes the missing-detail fallback after switching language", () => {
    const error = apiError(undefined, new LocalizedError((m) => m.shareUnavailable));
    expect(errorMessage(error, en, en.decryptError)).toBe(en.shareUnavailable);
    expect(errorMessage(error, uk, uk.decryptError)).toBe(uk.shareUnavailable);
  });

  it("re-renders an existing validation error in the selected language", () => {
    const error = new LocalizedError((m) => m.chooseOpens(10));
    expect(errorMessage(error, en, en.createError)).toBe("Choose between 1 and 10 opens.");
    expect(errorMessage(error, uk, uk.createError)).toBe(
      "Вкажіть кількість відкриттів від 1 до 10.",
    );
  });

  it("uses a localized fallback for browser and unexpected failures", () => {
    expect(errorMessage(new Error("Failed to fetch"), uk, uk.settingsFailed)).toBe(
      uk.settingsFailed,
    );
    expect(errorMessage("unexpected failure", en, en.createError)).toBe(en.createError);
  });
});
