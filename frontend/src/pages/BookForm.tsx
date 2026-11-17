import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api, yearOf, type BookInput } from "../lib/api";
import { useCanonicalPath, useLoad } from "../lib/hooks";
import { idFromSlug, toSlug } from "../lib/slug";
import { button, buttonLight, ErrorText, Field, input } from "../lib/ui";

const empty = { titel: "", autor: "", isbn: "", year: "", shelfId: "", praesenz: false };

/** Create (/books/new) and edit (/books/:slug/edit) share this form. */
export default function BookForm() {
  const { slug } = useParams();
  const editing = slug !== undefined;
  const id = idFromSlug(slug);
  const navigate = useNavigate();

  const shelves = useLoad(() => api.listShelves("", "name"), []);
  const book = useLoad(() => (id ? api.getBook(id) : Promise.resolve(undefined)), [id]);
  useCanonicalPath(book.data && `/books/${toSlug(book.data.ID, book.data.Titel)}/edit`);

  const [form, setForm] = useState(empty);
  const [file, setFile] = useState<File | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const set = (key: keyof typeof empty) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm((f) => ({ ...f, [key]: e.target.type === "checkbox" ? (e.target as HTMLInputElement).checked : e.target.value }));

  useEffect(() => {
    const b = book.data;
    if (b) {
      setForm({
        titel: b.Titel,
        autor: b.Autor,
        isbn: b.ISBN,
        year: yearOf(b.Jahr),
        shelfId: b.ShelfID ? String(b.ShelfID) : "",
        praesenz: b.Praesenzbestand,
      });
    }
  }, [book.data]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const input: BookInput = {
      Titel: form.titel.trim(),
      Autor: form.autor.trim(),
      ISBN: form.isbn.trim(),
      Jahr: form.year ? `${form.year.padStart(4, "0")}-01-01T00:00:00Z` : undefined,
      ShelfID: form.shelfId ? Number(form.shelfId) : null,
      Praesenzbestand: form.praesenz,
    };
    setSaving(true);
    try {
      const saved = id ? await api.updateBook(id, input) : await api.createBook(input);
      // The book needs an ID before it can get a file. If only the upload fails, the book
      // is saved anyway: go to its page and show the error there (re-submitting this
      // form would create the book twice).
      let uploadError: string | undefined;
      if (file) {
        try {
          await api.uploadFile(saved.ID, file);
        } catch (e) {
          uploadError = `Buch gespeichert, aber die Datei wurde nicht hochgeladen: ${(e as Error).message}`;
        }
      }
      navigate(`/books/${toSlug(saved.ID, saved.Titel)}`, { state: { uploadError } });
    } catch (e) {
      setError((e as Error).message);
      setSaving(false);
    }
  }

  if (editing && !id) return <p>Buch nicht gefunden.</p>;
  if (book.error) return <ErrorText error={book.error} />;
  if (editing && !book.data) return <p>Lädt …</p>;

  return (
    <form onSubmit={submit} className="max-w-md space-y-3">
      <h1 className="text-2xl font-semibold">{editing ? "Buch bearbeiten" : "Neues Buch"}</h1>
      <Field label="Titel *">
        <input className={input} value={form.titel} onChange={set("titel")} required autoFocus />
      </Field>
      <Field label="Autor">
        <input className={input} value={form.autor} onChange={set("autor")} />
      </Field>
      <Field label="ISBN">
        <input className={input} value={form.isbn} onChange={set("isbn")} />
      </Field>
      <Field label="Jahr">
        <input className={input} type="number" min={1} max={9999} value={form.year} onChange={set("year")} />
      </Field>
      <Field label="Regal">
        <select className={input} value={form.shelfId} onChange={set("shelfId")}>
          <option value="">— kein Regal —</option>
          {shelves.data?.map((s) => {
            const full = s.Books.length >= s.Limit && s.ID !== book.data?.ShelfID;
            return (
              <option key={s.ID} value={s.ID} disabled={full}>
                {s.ShelfName} ({s.Books.length}/{s.Limit}){full ? " – voll" : ""}
              </option>
            );
          })}
        </select>
      </Field>
      <label className="flex items-center gap-2">
        <input type="checkbox" checked={form.praesenz} onChange={set("praesenz")} />
        Präsenzbestand (nicht ausleihbar)
      </label>
      <Field label={book.data?.FileName ? `Datei ersetzen (jetzt: ${book.data.FileName})` : "Datei (PDF oder EPUB, max. 50 MB)"}>
        <input
          type="file"
          accept=".pdf,.epub,application/pdf,application/epub+zip"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          className="block text-sm"
        />
      </Field>
      <ErrorText error={error ?? shelves.error} />
      <div className="flex gap-2">
        <button className={button} disabled={saving}>
          {saving ? "Speichert …" : "Speichern"}
        </button>
        <Link to={book.data ? `/books/${toSlug(book.data.ID, book.data.Titel)}` : "/books"} className={buttonLight}>
          Abbrechen
        </Link>
      </div>
    </form>
  );
}
