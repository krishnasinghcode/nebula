import { useState } from "react";
import { api, type Image, type InstanceType, type Capacity, ApiError } from "../api/client";

export function CreateWizard({
  images,
  types,
  keys,
  capacity,
  onCreated,
}: {
  images: Image[];
  types: InstanceType[];
  keys: string[];
  capacity: Capacity | null;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [imageId, setImageId] = useState("");
  const [typeId, setTypeId] = useState("");
  const [keyId, setKeyId] = useState(keys[0] ?? "");
  const [userData, setUserData] = useState("");
  const [trusted, setTrusted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // keys load async; default to the first once available.
  const effectiveKey = keyId || keys[0] || "";
  const selectedType = types.find((t) => t.id === typeId);
  const host = capacity?.host;
  const fits =
    host?.schedulable_cpus != null && host?.schedulable_memory_mb != null && selectedType?.vcpu != null && selectedType?.memory_mb != null
      ? host.schedulable_cpus - (capacity?.used_cpus ?? 0) >= selectedType.vcpu &&
        host.schedulable_memory_mb - (capacity?.used_memory_mb ?? 0) >= selectedType.memory_mb
      : null;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await api.createInstance({
        name,
        image_id: imageId,
        type_id: typeId,
        key_id: effectiveKey,
        ...(userData.trim() ? { user_data: btoa(userData) } : {}),
      });
      setName("");
      setUserData("");
      setTrusted(false);
      onCreated();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const input =
    "rounded border border-zinc-700 bg-zinc-900 px-2 py-1.5 text-sm outline-none focus:border-blue-500";

  return (
    <form onSubmit={submit} className="space-y-3 rounded-lg border border-zinc-800 p-4" data-testid="create-wizard">
      <h2 className="font-semibold">Create instance</h2>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <label className="text-sm">
          Name
          <input
            className={input + " mt-1 w-full"}
            value={name}
            onChange={(e) => setName(e.target.value)}
            pattern="[a-z][a-z0-9\-]{2,30}"
            placeholder="web-1"
            required
            data-testid="input-name"
          />
        </label>
        <label className="text-sm">
          Image
          <select className={input + " mt-1 w-full"} value={imageId} onChange={(e) => setImageId(e.target.value)} required data-testid="input-image">
            <option value="">—</option>
            {images.map((i) => (
              <option key={i.id} value={i.id}>
                {i.name}
              </option>
            ))}
          </select>
        </label>
        <label className="text-sm">
          Type
          <select className={input + " mt-1 w-full"} value={typeId} onChange={(e) => setTypeId(e.target.value)} required data-testid="input-type">
            <option value="">—</option>
            {types.map((t) => (
              <option key={t.id} value={t.id}>
                {t.description} ({t.id})
              </option>
            ))}
          </select>
        </label>
        <label className="text-sm">
          Key
          <select className={input + " mt-1 w-full"} value={effectiveKey} onChange={(e) => setKeyId(e.target.value)} required data-testid="input-key">
            {keys.length === 0 && <option value="">no keys registered</option>}
            {keys.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label className="block text-sm">
        User data (optional, shell script)
        <textarea
          className={input + " mt-1 w-full font-mono"}
          rows={3}
          value={userData}
          onChange={(e) => setUserData(e.target.value)}
          placeholder="#!/bin/sh"
          data-testid="input-userdata"
        />
      </label>
      {fits === false && (
        <p className="text-sm text-yellow-400" data-testid="capacity-warning">
          Selected type does not fit the current schedulable capacity; creation may be rejected.
        </p>
      )}
      {capacity && (
        <p className="text-xs text-zinc-500">
          Capacity: {capacity.used_cpus}/{capacity.host!.schedulable_cpus} vCPU ·{" "}
          {capacity.used_memory_mb}/{capacity.host!.schedulable_memory_mb} MB · {capacity.instances}/
          {capacity.max_instances} instances
        </p>
      )}
      {/* Trust-model acknowledgement (security.md §1) — required per create. */}
      <label className="flex items-start gap-2 text-sm text-zinc-300" data-testid="trust-ack">
        <input type="checkbox" checked={trusted} onChange={(e) => setTrusted(e.target.checked)} className="mt-0.5" required />
        <span>
          I understand Orbit instances are containers sharing my host kernel — not a secure
          multi-tenant hypervisor.
        </span>
      </label>
      {error && <p className="text-sm text-red-400" data-testid="create-error">{error}</p>}
      <button
        type="submit"
        disabled={busy || !trusted || !effectiveKey}
        className="rounded bg-blue-600 px-3 py-1.5 text-sm font-medium hover:bg-blue-500 disabled:opacity-50"
        data-testid="create-submit"
      >
        {busy ? "Creating…" : "Create"}
      </button>
    </form>
  );
}
