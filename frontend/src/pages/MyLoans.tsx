import { useState } from "react";
import { Link } from "react-router";
import { api, formatDate, type Loan } from "../lib/api";
import { useLoad } from "../lib/hooks";
import { toSlug } from "../lib/slug";
import { button, ErrorText, link } from "../lib/ui";
import { useUser } from "../lib/user";

function BookLink({ loan }: { loan: Loan }) {
  if (!loan.Book) return <span className="text-gray-600">Gelöschtes Buch</span>;
  return (
    <Link to={`/books/${toSlug(loan.Book.ID, loan.Book.Titel)}`} className={link}>
      {loan.Book.Titel}
    </Link>
  );
}

export default function MyLoans() {
  const { person } = useUser();
  const me = useLoad(() => api.getPerson(person.ID), [person.ID]);
  const [error, setError] = useState<string>();

  async function giveBack(bookId: number) {
    setError(undefined);
    try {
      await api.giveBack(bookId);
      me.reload();
    } catch (e) {
      setError((e as Error).message);
    }
  }

  const loans = me.data?.Loans ?? [];
  const open = loans.filter((l) => !l.ReturnedAt);
  const history = loans.filter((l) => l.ReturnedAt).reverse(); // newest first

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Meine Ausleihen</h1>
      <ErrorText error={error ?? me.error} />
      {!me.data && !me.error && <p>Lädt …</p>}

      {me.data && (
        <>
          <section className="space-y-2">
            <h2 className="font-semibold">Gerade ausgeliehen</h2>
            {open.length === 0 && <p className="text-gray-600">Du hast gerade keine Bücher ausgeliehen.</p>}
            <ul className="divide-y divide-gray-200">
              {open.map((l) => (
                <li key={l.ID} className="flex items-center justify-between py-2">
                  <span>
                    <BookLink loan={l} /> <span className="text-gray-600">seit {formatDate(l.LoanedAt)}</span>
                  </span>
                  <button className={button} onClick={() => giveBack(l.BookID)}>
                    Zurückgeben
                  </button>
                </li>
              ))}
            </ul>
          </section>

          <section className="space-y-2">
            <h2 className="font-semibold">Verlauf</h2>
            {history.length === 0 && <p className="text-gray-600">Noch nichts zurückgegeben.</p>}
            <ul className="divide-y divide-gray-200">
              {history.map((l) => (
                <li key={l.ID} className="py-2">
                  <BookLink loan={l} />{" "}
                  <span className="text-gray-600">
                    {formatDate(l.LoanedAt)} – {formatDate(l.ReturnedAt!)}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </div>
  );
}
