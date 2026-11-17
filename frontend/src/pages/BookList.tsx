import { Link } from "react-router";
import { api, yearOf } from "../lib/api";
import { useLoad } from "../lib/hooks";
import { SearchSort, useSearchSort } from "../lib/search";
import { toSlug } from "../lib/slug";
import { button, ErrorText, link } from "../lib/ui";

const sorts: [string, string][] = [
  ["titel", "Titel A–Z"],
  ["-titel", "Titel Z–A"],
  ["autor", "Autor A–Z"],
  ["-jahr", "Jahr: neueste"],
  ["jahr", "Jahr: älteste"],
  ["-created", "Zuletzt angelegt"],
];

export default function BookList() {
  const search = useSearchSort("titel");
  const { data: books, error } = useLoad(() => api.listBooks(search.q, search.sort), [search.q, search.sort]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Bücher</h1>
        <Link to="/books/new" className={button}>
          Neues Buch
        </Link>
      </div>
      <SearchSort {...search} options={sorts} />
      <ErrorText error={error} />
      {books?.length === 0 && <p className="text-gray-600">Keine Bücher gefunden.</p>}
      <ul className="divide-y divide-gray-200">
        {books?.map((b) => (
          <li key={b.ID} className="py-2">
            <Link to={`/books/${toSlug(b.ID, b.Titel)}`} className={link}>
              {b.Titel}
            </Link>{" "}
            <span className="text-gray-600">{[b.Autor, yearOf(b.Jahr)].filter(Boolean).join(", ")}</span>
            {b.Praesenzbestand && <span className="ml-2 rounded bg-amber-100 px-1 text-xs">Präsenzbestand</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}
