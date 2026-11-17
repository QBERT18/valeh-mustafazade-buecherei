// Types mirror the Go structs (gorm.Model fields + the model's own fields).

export type Shelf = {
  ID: number;
  ShelfName: string;
  Limit: number;
  Books: Book[];
};

export type Book = {
  ID: number;
  Titel: string;
  Autor: string;
  ISBN: string;
  Jahr: string; // RFC 3339; year 1 = not set
  Praesenzbestand: boolean;
  ShelfID: number | null;
  Shelf?: Shelf | null;
  FileName: string;
  FileSize: number;
};

export type Loan = {
  ID: number;
  BookID: number;
  Book?: Book | null;
  PersonID: number;
  LoanedAt: string;
  ReturnedAt: string | null;
};

export type Person = {
  ID: number;
  Name: string;
  Loans: Loan[];
};

export type BookInput = Pick<Book, "Titel" | "Autor" | "ISBN" | "Praesenzbestand" | "ShelfID"> & { Jahr?: string };
export type ShelfInput = Pick<Shelf, "ShelfName" | "Limit">;

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  // FormData sets its own multipart Content-Type (with boundary), so leave the header out.
  const isForm = body instanceof FormData;
  const res = await fetch(`/api${path}`, {
    method,
    headers: body && !isForm ? { "Content-Type": "application/json" } : undefined,
    body: isForm ? body : body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const data = await res.json().catch(() => null);
    throw new ApiError(res.status, data?.error ?? `HTTP ${res.status}`);
  }
  return res.status === 204 ? (undefined as T) : res.json();
}

const list = (path: string, q: string, sort: string) => `${path}?${new URLSearchParams({ q, sort })}`;

export const api = {
  listBooks: (q = "", sort = "") => request<Book[]>("GET", list("/books", q, sort)),
  getBook: (id: number) => request<Book>("GET", `/books/${id}`),
  createBook: (b: BookInput) => request<Book>("POST", "/books", b),
  updateBook: (id: number, b: BookInput) => request<Book>("PUT", `/books/${id}`, b),
  deleteBook: (id: number) => request<void>("DELETE", `/books/${id}`),
  fileUrl: (id: number) => `/api/books/${id}/file`,
  uploadFile: (id: number, file: File) => {
    const form = new FormData();
    form.append("file", file);
    return request<Book>("PUT", `/books/${id}/file`, form);
  },

  listShelves: (q = "", sort = "") => request<Shelf[]>("GET", list("/shelves", q, sort)),
  getShelf: (id: number) => request<Shelf>("GET", `/shelves/${id}`),
  createShelf: (s: ShelfInput) => request<Shelf>("POST", "/shelves", s),
  updateShelf: (id: number, s: ShelfInput) => request<Shelf>("PUT", `/shelves/${id}`, s),
  deleteShelf: (id: number) => request<void>("DELETE", `/shelves/${id}`),

  getPerson: (id: number) => request<Person>("GET", `/people/${id}`),
  createPerson: (Name: string) => request<Person>("POST", "/people", { Name }),

  listLoans: () => request<Loan[]>("GET", "/loans"),
  lend: (bookId: number, PersonID: number) => request<Loan>("POST", `/books/${bookId}/loan`, { PersonID }),
  giveBack: (bookId: number) => request<Loan>("POST", `/books/${bookId}/return`),
};

/** Year from the API date, or "" if not set (Go's zero date is year 1). */
export function yearOf(jahr: string): string {
  const y = new Date(jahr).getUTCFullYear();
  return y > 1 ? String(y) : "";
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("de-DE");
}
