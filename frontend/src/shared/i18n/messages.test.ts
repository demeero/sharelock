import { describe, expect, it } from "vitest";

import { en, uk } from "./messages";

describe("Ukrainian messages", () => {
  it.each([
    [0, "відкриттів", "елементів", "разів"],
    [1, "відкриття", "елемент", "раз"],
    [2, "відкриття", "елементи", "рази"],
    [5, "відкриттів", "елементів", "разів"],
    [11, "відкриттів", "елементів", "разів"],
    [21, "відкриття", "елемент", "раз"],
    [22, "відкриття", "елементи", "рази"],
    [25, "відкриттів", "елементів", "разів"],
  ])("formats count %i", (count, opens, items, times) => {
    expect(uk.opens(count)).toBe(`${count} ${opens}`);
    expect(uk.payloads(count)).toBe(`${count} ${items}`);
    expect(uk.itemCount(count)).toBe(`${count} ${items.toUpperCase()}`);
    expect(uk.remainingOpens(count)).toBe(
      `До видалення цей секрет можна відкрити ще ${count} ${times}.`,
    );
  });

  it.each([
    [1, "1 секунда"],
    [2, "2 секунди"],
    [5, "5 секунд"],
    [11, "11 секунд"],
    [21, "21 секунда"],
    [60, "1 хвилина"],
    [120, "2 хвилини"],
    [300, "5 хвилин"],
    [3600, "1 година"],
    [7200, "2 години"],
    [18000, "5 годин"],
    [86400, "1 день"],
    [172800, "2 дні"],
    [604800, "7 днів"],
    [1814400, "21 день"],
  ])("formats duration %i", (seconds, expected) => {
    expect(uk.duration(seconds)).toBe(expected);
  });

  it("preserves English singular and plural messages", () => {
    expect(en.opens(1)).toBe("1 open");
    expect(en.opens(5)).toBe("5 opens");
    expect(en.duration(86400)).toBe("1 day");
    expect(en.duration(604800)).toBe("7 days");
  });
});
