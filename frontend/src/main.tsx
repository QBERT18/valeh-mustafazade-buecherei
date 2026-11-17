import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, NavLink, Outlet, Route, Routes } from "react-router";
import "./index.css";
import { UserGate, useUser } from "./lib/user";
import BookDetail from "./pages/BookDetail";
import BookForm from "./pages/BookForm";
import BookList from "./pages/BookList";
import MyLoans from "./pages/MyLoans";
import ShelfDetail from "./pages/ShelfDetail";
import ShelfForm from "./pages/ShelfForm";
import ShelfList from "./pages/ShelfList";

function Layout() {
  const { person, logout } = useUser();
  const nav = ({ isActive }: { isActive: boolean }) => (isActive ? "font-semibold" : "text-gray-600 hover:text-gray-900");
  return (
    <div className="mx-auto max-w-3xl p-4">
      <header className="mb-6 flex flex-wrap items-center gap-4 border-b border-gray-200 pb-3">
        <span className="text-lg font-bold">Bücherei</span>
        <NavLink to="/books" className={nav}>
          Bücher
        </NavLink>
        <NavLink to="/shelves" className={nav}>
          Regale
        </NavLink>
        <NavLink to="/me" className={nav}>
          Meine Ausleihen
        </NavLink>
        <span className="ml-auto text-sm text-gray-600">
          {person.Name} ·{" "}
          <button onClick={logout} className="hover:underline">
            Abmelden
          </button>
        </span>
      </header>
      <Outlet />
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <UserGate>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Navigate to="/books" replace />} />
            <Route path="books" element={<BookList />} />
            <Route path="books/new" element={<BookForm />} />
            <Route path="books/:slug" element={<BookDetail />} />
            <Route path="books/:slug/edit" element={<BookForm />} />
            <Route path="shelves" element={<ShelfList />} />
            <Route path="shelves/new" element={<ShelfForm />} />
            <Route path="shelves/:slug" element={<ShelfDetail />} />
            <Route path="shelves/:slug/edit" element={<ShelfForm />} />
            <Route path="me" element={<MyLoans />} />
            <Route path="*" element={<p>Seite nicht gefunden.</p>} />
          </Route>
        </Routes>
      </UserGate>
    </BrowserRouter>
  </StrictMode>,
);
