import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api } from "../lib/api";
import { useCanonicalPath, useLoad } from "../lib/hooks";
import { idFromSlug, toSlug } from "../lib/slug";
import { button, buttonLight, ErrorText, Field, input } from "../lib/ui";

/** Create (/shelves/new) and edit (/shelves/:slug/edit) share this form. */
export default function ShelfForm() {
  const { slug } = useParams();
  const editing = slug !== undefined;
  const id = idFromSlug(slug);
  const navigate = useNavigate();

  const shelf = useLoad(() => (id ? api.getShelf(id) : Promise.resolve(undefined)), [id]);
  useCanonicalPath(shelf.data && `/shelves/${toSlug(shelf.data.ID, shelf.data.ShelfName)}/edit`);

  const [name, setName] = useState("");
  const [limit, setLimit] = useState("10");
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (shelf.data) {
      setName(shelf.data.ShelfName);
      setLimit(String(shelf.data.Limit));
    }
  }, [shelf.data]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const input = { ShelfName: name.trim(), Limit: Number(limit) };
    try {
      const saved = id ? await api.updateShelf(id, input) : await api.createShelf(input);
      navigate(`/shelves/${toSlug(saved.ID, saved.ShelfName)}`);
    } catch (e) {
      setError((e as Error).message);
    }
  }

  if (editing && !id) return <p>Regal nicht gefunden.</p>;
  if (shelf.error) return <ErrorText error={shelf.error} />;
  if (editing && !shelf.data) return <p>Lädt …</p>;

  return (
    <form onSubmit={submit} className="max-w-md space-y-3">
      <h1 className="text-2xl font-semibold">{editing ? "Regal bearbeiten" : "Neues Regal"}</h1>
      <Field label="Name *">
        <input className={input} value={name} onChange={(e) => setName(e.target.value)} required autoFocus />
      </Field>
      <Field label="Platz für wie viele Bücher? *">
        <input className={input} type="number" min={1} value={limit} onChange={(e) => setLimit(e.target.value)} required />
      </Field>
      <ErrorText error={error} />
      <div className="flex gap-2">
        <button className={button}>Speichern</button>
        <Link
          to={shelf.data ? `/shelves/${toSlug(shelf.data.ID, shelf.data.ShelfName)}` : "/shelves"}
          className={buttonLight}
        >
          Abbrechen
        </Link>
      </div>
    </form>
  );
}
