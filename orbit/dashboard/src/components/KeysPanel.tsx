import { useState } from "react";
import { api } from "../api/client";
import { usePoll } from "../hooks/usePoll";
import { Info, InfoList } from "./Info";
import { toast } from "./Toast";

export function KeysPanel() {
  const [name, setName] = useState("");
  const [pub, setPub] = useState("");
  const { data: keys, refresh } = usePoll(api.listKeys, 5000);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.createKey(name, pub);
      setName("");
      setPub("");
      refresh();
      toast(`Key "${name}" registered`, "success");
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err));
    }
  }

  async function remove(name: string) {
    if (!confirm(`Delete key "${name}"?`)) return;
    try {
      await api.deleteKey(name);
      refresh();
      toast(`Key "${name}" deleted`, "success");
    } catch (err) {
      toast(err instanceof Error ? err.message : String(err));
    }
  }

  const input = "console-input";

  return (
    <section className="rounded-lg border border-line bg-panel p-4" data-testid="keys-panel">
      <h2 className="mb-2 flex items-center text-sm font-semibold uppercase tracking-wider">
        SSH keys
        <Info brief="Public keys that let their holder log in to your instances.">
          <InfoList
            items={[
              ["Name", <>short label for the key (e.g. dev, laptop).</>],
              ["Public key", <>paste the contents of your public key file (e.g. ~/.ssh/id_ed25519.pub). The private half never leaves your machine.</>],
              ["Usage", <>key registered here lets its holder SSH into instances as the orbit user; CLI-created keys additionally work with <code>orbit ssh</code>.</>],
              ["Delete", <>removes the key — only possible while no instance still references it.</>],
            ]}
          />
        </Info>
      </h2>
      <ul className="mb-3 space-y-1 text-sm">
        {(keys ?? []).map((k) => (
          <li key={k.name} className="flex items-center justify-between gap-2">
            <span>
              <span className="font-medium">{k.name}</span>{" "}
              <span className="font-mono text-xs text-dim">{k.fingerprint}</span>
            </span>
            <button className="text-xs text-bad hover:opacity-75" onClick={() => remove(k.name!)}>
              delete
            </button>
          </li>
        ))}
        {(keys ?? []).length === 0 && <li className="text-dim">No keys registered.</li>}
      </ul>
      <form onSubmit={create} className="space-y-2">
        <input className={input + " w-40"} value={name} onChange={(e) => setName(e.target.value)} placeholder="name" pattern="[a-z][a-z0-9\-]{2,30}" required title="Short label for the key (e.g. dev, laptop)" data-testid="key-name" />
        <textarea className={input + " w-full font-mono text-xs"} rows={2} value={pub} onChange={(e) => setPub(e.target.value)} placeholder="ssh-ed25519 AAAA…" required title="Contents of your public key file (e.g. ~/.ssh/id_ed25519.pub). The private key stays on your machine." data-testid="key-pub" />
        <button className="btn-primary">Add key</button>
      </form>
    </section>
  );
}
