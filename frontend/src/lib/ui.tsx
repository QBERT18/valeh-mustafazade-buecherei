import type { ReactNode } from "react";

export const input = "w-full rounded border border-gray-300 px-2 py-1";
export const button = "rounded bg-gray-900 px-3 py-1 text-white hover:bg-gray-700 disabled:opacity-50";
export const buttonLight = "rounded border border-gray-300 px-3 py-1 hover:bg-gray-100";
export const buttonDanger = "rounded border border-red-300 px-3 py-1 text-red-700 hover:bg-red-50";
export const link = "text-blue-700 hover:underline";

export function ErrorText({ error }: { error?: string }) {
  return error ? <p className="text-red-700">{error}</p> : null;
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm text-gray-600">{label}</span>
      {children}
    </label>
  );
}
