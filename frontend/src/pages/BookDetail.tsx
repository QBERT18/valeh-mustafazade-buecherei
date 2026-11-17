import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api, formatDate, yearOf } from "../lib/api";
import { useCanonicalPath, useLoad } from "../lib/hooks";
import { idFromSlug, toSlug } from "../lib/slug";
import { button, buttonDanger, buttonLight, ErrorText, link } from "../lib/ui";
import { useUser } from "../lib/user";

export default function BookDetail() {
  const id = idFromSlug(useParams().slug);
  const { person } = useUser();
  const navigate = useNavigate();

  const book = useLoad(() => (id ? api.getBook(id) : Promise.reject(new Error("Buch nicht gefunden."))), [id]);
  const loans = useLoad(() => api.listLoans(), [id]);
  useCanonicalPath(book.data && `/books/${toSlug(book.data.ID, book.data.Titel)}`);
  const [error, setError] = useState<string>();

  const b = book.data;
  if (book.error) return <ErrorText error={book.error} />;
  if (!b) return <p>Lädt …</p>;

  const openLoan = loans.data?.find((l) => l.BookID === b.ID && !l.ReturnedAt);

  async function run(action: () => Promise<unknown>) {
    setError(undefined);
    try {
      await action();
      loans.reload();
    } catch (e) {
      setError((e as Error).message);
    }
  }

  async function remove() {
    if (!confirm(`„${b!.Titel}“ wirklich löschen?`)) return;
    await run(async () => {
      await api.deleteBook(b!.ID);
      navigate("/books");
    });
  }

  const rows: [string, React.ReactNode][] = [
    ["Autor", b.Autor || "–"],
    ["ISBN", b.ISBN || "–"],
    ["Jahr", yearOf(b.Jahr) || "–"],
    [
      "Regal",
      b.Shelf ? (
        <Link to={`/shelves/${toSlug(b.Shelf.ID, b.Shelf.ShelfName)}`} className={link}>
          {b.Shelf.ShelfName}
        </Link>
      ) : (
        "–"
      ),
    ],
    [
      "Datei",
      b.FileName ? (
        <a href={api.fileUrl(b.ID)} className={link}>
          {b.FileName} ({Math.ceil(b.FileSize / 1024)} KB)
        </a>
      ) : (
        "–"
      ),
    ],
  ];

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">{b.Titel}</h1>
      <dl className="grid grid-cols-[8rem_1fr] gap-y-1">
        {rows.map(([label, value]) => (
          <div key={label} className="contents">
            <dt className="text-gray-600">{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>

      <div className="rounded border border-gray-200 p-3">
        {b.Praesenzbestand ? (
          <p>Präsenzbestand – kann nur in der Bücherei gelesen werden.</p>
        ) : !loans.data ? (
          <p>Lädt …</p>
        ) : !openLoan ? (
          <div className="flex items-center justify-between">
            <span>Verfügbar</span>
            <button className={button} onClick={() => run(() => api.lend(b.ID, person.ID))}>
              Ausleihen
            </button>
          </div>
        ) : openLoan.PersonID === person.ID ? (
          <div className="flex items-center justify-between">
            <span>Von dir ausgeliehen seit {formatDate(openLoan.LoanedAt)}</span>
            <button className={button} onClick={() => run(() => api.giveBack(b.ID))}>
              Zurückgeben
            </button>
          </div>
        ) : (
          <p>Ausgeliehen seit {formatDate(openLoan.LoanedAt)}</p>
        )}
      </div>

      <ErrorText error={error ?? loans.error} />
      <div className="flex gap-2">
        <Link to={`/books/${toSlug(b.ID, b.Titel)}/edit`} className={buttonLight}>
          Bearbeiten
        </Link>
        <button className={buttonDanger} onClick={remove}>
          Löschen
        </button>
        <Link to="/books" className={`${link} ml-auto self-center`}>
          ← Alle Bücher
        </Link>
      </div>
    </div>
  );
}
