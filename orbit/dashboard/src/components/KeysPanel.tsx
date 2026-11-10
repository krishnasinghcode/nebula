import { useState } from "react";
import { api } from "../api/client";
import { usePoll } from "../hooks/usePoll";

export function KeysPanel() {
  const [name, setName] = useState("");
  const [pub, setPub] = useState("");
  const [error, setError] = useState<string | null>(null);
  const { data: keys, refresh } = usePoll(api.listKeys, 5000);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.createKey(name, pub);
      setName("");
      setPub("");
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function remove(name: string) {
    if (!confirm(`Delete key "${name}"?`)) return;
    try {
      await api.deleteKey(name);
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  const input = "rounded border border-zinc-700 bg-zinc-900 px-2 py-1.5 text-sm outline-none focus:border-blue-500";

  return (
    <section className="rounded-lg border border-zinc-800 p-4" data-testid="keys-panel">
      <h2 className="mb-2 font-semibold">SSH keys</h2>
      <ul className="mb-3 space-y-1 text-sm">
        {(keys ?? []).map((k) => (
          <li key={k.name} className="flex items-center justify-between gap-2">
            <span>
              <span className="font-medium">{k.name}</span> <span className="text-zinc-500">{k.fingerprint}</span>
            </span>
            <button className="text-xs text-red-400 hover:text-red-300" onClick={() => remove(k.name!)}>
              delete
            </button>
          </li>
        ))}
        {(keys ?? []).length === 0 && <li className="text-zinc-500">No keys registered.</li>}
      </ul>
      <form onSubmit={create} className="space-y-2">
        <input className={input + " w-40"} value={name} onChange={(e) => setName(e.target.value)} placeholder="name" pattern="[a-z][a-z0-9-]{2,30}" required data-testid="key-name" />
        <textarea className={input + " w-full font-mono"} rows={2} value={pub} onChange={(e) => setPub(e.target.value)} placeholder="ssh-ed25519 AAAA…" required data-testid="key-pub" />
        {error && <p className="text-sm text-red-400">{error}</p>}
        <button className="rounded bg-blue-600 px-3 py-1.5 text-sm hover:bg-blue-500">Add key</button>
      </form>
    </section>
  );
}
