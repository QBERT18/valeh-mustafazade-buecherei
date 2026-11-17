import { useSearchParams } from "react-router";
import { input } from "./ui";

/** Search term and sort order live in the URL (?q=&sort=), so a list can be bookmarked. */
export function useSearchSort(defaultSort: string) {
  const [params, setParams] = useSearchParams();
  const set = (key: string, value: string) =>
    setParams(
      (p) => {
        if (value) p.set(key, value);
        else p.delete(key);
        return p;
      },
      { replace: true },
    );
  return {
    q: params.get("q") ?? "",
    sort: params.get("sort") ?? defaultSort,
    setQ: (v: string) => set("q", v),
    setSort: (v: string) => set("sort", v),
  };
}

type Props = ReturnType<typeof useSearchSort> & { options: [value: string, label: string][] };

export function SearchSort({ q, sort, setQ, setSort, options }: Props) {
  return (
    <div className="flex gap-2">
      <input className={input} type="search" placeholder="Suchen …" value={q} onChange={(e) => setQ(e.target.value)} />
      <select className="rounded border border-gray-300 px-2" value={sort} onChange={(e) => setSort(e.target.value)}>
        {options.map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
    </div>
  );
}
