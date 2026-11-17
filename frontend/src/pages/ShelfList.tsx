import { Link } from "react-router";
import { api } from "../lib/api";
import { useLoad } from "../lib/hooks";
import { SearchSort, useSearchSort } from "../lib/search";
import { toSlug } from "../lib/slug";
import { button, ErrorText, link } from "../lib/ui";

const sorts: [string, string][] = [
  ["name", "Name A–Z"],
  ["-name", "Name Z–A"],
  ["-limit", "Größte zuerst"],
  ["limit", "Kleinste zuerst"],
];

export default function ShelfList() {
  const search = useSearchSort("name");
  const { data: shelves, error } = useLoad(() => api.listShelves(search.q, search.sort), [search.q, search.sort]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Regale</h1>
        <Link to="/shelves/new" className={button}>
          Neues Regal
        </Link>
      </div>
      <SearchSort {...search} options={sorts} />
      <ErrorText error={error} />
      {shelves?.length === 0 && <p className="text-gray-600">Keine Regale gefunden.</p>}
      <ul className="divide-y divide-gray-200">
        {shelves?.map((s) => (
          <li key={s.ID} className="flex justify-between py-2">
            <Link to={`/shelves/${toSlug(s.ID, s.ShelfName)}`} className={link}>
              {s.ShelfName}
            </Link>
            <span className="text-gray-600">
              {s.Books.length} / {s.Limit} Bücher
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
