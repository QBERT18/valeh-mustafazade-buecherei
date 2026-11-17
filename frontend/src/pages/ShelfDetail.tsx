import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api, yearOf } from "../lib/api";
import { useCanonicalPath, useLoad } from "../lib/hooks";
import { idFromSlug, toSlug } from "../lib/slug";
import { buttonDanger, buttonLight, ErrorText, link } from "../lib/ui";

export default function ShelfDetail() {
  const id = idFromSlug(useParams().slug);
  const navigate = useNavigate();
  const shelf = useLoad(() => (id ? api.getShelf(id) : Promise.reject(new Error("Regal nicht gefunden."))), [id]);
  useCanonicalPath(shelf.data && `/shelves/${toSlug(shelf.data.ID, shelf.data.ShelfName)}`);
  const [error, setError] = useState<string>();

  const s = shelf.data;
  if (shelf.error) return <ErrorText error={shelf.error} />;
  if (!s) return <p>Lädt …</p>;

  async function remove() {
    if (!confirm(`Regal „${s!.ShelfName}“ löschen? Die Bücher bleiben erhalten und stehen dann auf keinem Regal.`)) return;
    try {
      await api.deleteShelf(s!.ID);
      navigate("/shelves");
    } catch (e) {
      setError((e as Error).message);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">{s.ShelfName}</h1>
      <p className="text-gray-600">
        {s.Books.length} von {s.Limit} Plätzen belegt
      </p>
      {s.Books.length === 0 ? (
        <p className="text-gray-600">Noch keine Bücher auf diesem Regal.</p>
      ) : (
        <ul className="divide-y divide-gray-200">
          {s.Books.map((b) => (
            <li key={b.ID} className="py-2">
              <Link to={`/books/${toSlug(b.ID, b.Titel)}`} className={link}>
                {b.Titel}
              </Link>{" "}
              <span className="text-gray-600">{[b.Autor, yearOf(b.Jahr)].filter(Boolean).join(", ")}</span>
            </li>
          ))}
        </ul>
      )}
      <ErrorText error={error} />
      <div className="flex gap-2">
        <Link to={`/shelves/${toSlug(s.ID, s.ShelfName)}/edit`} className={buttonLight}>
          Bearbeiten
        </Link>
        <button className={buttonDanger} onClick={remove}>
          Löschen
        </button>
        <Link to="/shelves" className={`${link} ml-auto self-center`}>
          ← Alle Regale
        </Link>
      </div>
    </div>
  );
}
