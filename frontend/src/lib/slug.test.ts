import { describe, expect, it } from "vitest";
import { idFromSlug, slugify, toSlug } from "./slug";

describe("slugify", () => {
  it.each([
    ["Faust. Eine Tragödie", "faust-eine-tragoedie"],
    ["Straße der Ölmühle", "strasse-der-oelmuehle"],
    ["ÄÖÜ", "aeoeue"],
    ["Café Crème", "cafe-creme"],
    ["  --Hallo,   Welt!--  ", "hallo-welt"],
    ["100% Rabatt", "100-rabatt"],
    ["!!!", ""],
  ])("%s → %s", (input, want) => {
    expect(slugify(input)).toBe(want);
  });
});

describe("toSlug / idFromSlug", () => {
  it("puts the id first and reads it back", () => {
    expect(toSlug(12, "Effi Briest")).toBe("12-effi-briest");
    expect(idFromSlug("12-effi-briest")).toBe(12);
  });

  it("falls back to the bare id when the text has no letters", () => {
    expect(toSlug(3, "!!!")).toBe("3");
    expect(idFromSlug("3")).toBe(3);
  });

  it("rejects slugs without a leading id", () => {
    expect(idFromSlug("effi-briest")).toBeNull();
    expect(idFromSlug("12abc")).toBeNull();
    expect(idFromSlug(undefined)).toBeNull();
  });
});
