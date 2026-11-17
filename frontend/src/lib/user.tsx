import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { api, ApiError, type Person } from "./api";
import { button, ErrorText, input } from "./ui";

// There is no login: the browser remembers which person it is.
const KEY = "buecherei.personId";

function readId(): number | null {
  try {
    const v = Number(localStorage.getItem(KEY));
    return v > 0 ? v : null;
  } catch {
    return null;
  }
}

function writeId(id: number | null) {
  try {
    if (id) localStorage.setItem(KEY, String(id));
    else localStorage.removeItem(KEY);
  } catch {
    // storage blocked: the user just has to enter the name again next time
  }
}

type User = { person: Person; logout: () => void };
const UserContext = createContext<User | null>(null);

export function useUser(): User {
  const user = useContext(UserContext);
  if (!user) throw new Error("useUser outside of <UserGate>");
  return user;
}

/** Shows a name form until the browser has a person, then renders the app. */
export function UserGate({ children }: { children: ReactNode }) {
  const [id, setId] = useState(readId);
  const [person, setPerson] = useState<Person | null>(null);
  const [name, setName] = useState("");
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (id == null) return;
    api.getPerson(id).then(setPerson, (e) => {
      if (e instanceof ApiError && e.status === 404) logout(); // person was deleted
      else setError(e.message);
    });
  }, [id]);

  function logout() {
    writeId(null);
    setId(null);
    setPerson(null);
  }

  async function register(e: React.FormEvent) {
    e.preventDefault();
    try {
      const p = await api.createPerson(name.trim());
      writeId(p.ID);
      setId(p.ID);
      setPerson(p);
    } catch (e) {
      setError((e as Error).message);
    }
  }

  if (id == null) {
    return (
      <form onSubmit={register} className="mx-auto mt-24 max-w-sm space-y-3 p-4">
        <h1 className="text-xl font-semibold">Willkommen in der Bücherei</h1>
        <p className="text-gray-600">Wie heißt du?</p>
        <input className={input} value={name} onChange={(e) => setName(e.target.value)} autoFocus required />
        <button className={button} disabled={!name.trim()}>
          Los geht's
        </button>
        <ErrorText error={error} />
      </form>
    );
  }
  if (!person) return <div className="p-4">{error ? <ErrorText error={error} /> : "Lädt …"}</div>;

  return <UserContext value={{ person, logout }}>{children}</UserContext>;
}
