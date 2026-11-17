const umlauts: Record<string, string> = { ä: "ae", ö: "oe", ü: "ue", Ä: "ae", Ö: "oe", Ü: "ue", ß: "ss" };

/** "Faust. Eine Tragödie" → "faust-eine-tragoedie" */
export function slugify(text: string): string {
  return text
    .replace(/[äöüÄÖÜß]/g, (c) => umlauts[c])
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "") // other accents: é → e
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

/** URL segment: id first, so the page can find the record even if the name changed. */
export function toSlug(id: number, text: string): string {
  const s = slugify(text);
  return s ? `${id}-${s}` : String(id);
}

export function idFromSlug(slug: string | undefined): number | null {
  const m = /^(\d+)(-|$)/.exec(slug ?? "");
  return m ? Number(m[1]) : null;
}
