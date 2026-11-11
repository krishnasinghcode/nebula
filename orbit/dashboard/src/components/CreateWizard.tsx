import { useState } from "react";
import { api, type Image, type InstanceType, type Capacity, ApiError } from "../api/client";
import { Info, InfoList } from "./Info";
import { toast } from "./Toast";

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
        ...(userData.trim() ? { user_data: userData } : {}),
      });
      setName("");
      setUserData("");
      setTrusted(false);
      onCreated();
      toast(`Instance "${name}" is being created`, "success");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
      toast(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3 rounded-lg border border-line bg-panel p-4" data-testid="create-wizard">
      <h2 className="flex items-center text-sm font-semibold uppercase tracking-wider">
        Create instance
        <Info
          brief="Provision a new Orbit Instance: an emulated compute node with its own SSH access, private IP and ports."
        >
          <InfoList
            items={[
              ["Name", <>unique lowercase label among live instances (e.g. web-1); also the idempotency anchor for double-clicks.</>],
              ["Image", <>the OS image the instance boots from (e.g. Ubuntu 24.04).</>],
              ["Type", <>hardware shape: vCPU count and memory (like an EC2 instance type). Must fit the remaining schedulable capacity.</>],
              ["Key", <>SSH key whose public half is installed in the instance. Imported keys: connect with your own <code>ssh -i</code>. CLI-created keys also work with <code>orbit ssh</code>.</>],
              ["User data", <>optional shell script that runs once inside the instance on first boot (like EC2 user data). Must start with a shebang, e.g. <code>#!/bin/sh</code>.</>],
              ["Trust ack", <>instances are containers sharing your host kernel — anyone with root inside one is effectively root on your host. Only run workloads you trust.</>],
            ]}
          />
        </Info>
      </h2>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <label className="text-sm">
          Name
          <input
            className="console-input mt-1 w-full"
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
          <select className="console-input mt-1 w-full" value={imageId} onChange={(e) => setImageId(e.target.value)} required data-testid="input-image">
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
          <select className="console-input mt-1 w-full" value={typeId} onChange={(e) => setTypeId(e.target.value)} required data-testid="input-type">
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
          <select className="console-input mt-1 w-full" value={effectiveKey} onChange={(e) => setKeyId(e.target.value)} required data-testid="input-key">
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
          className="console-input mt-1 w-full font-mono text-xs"
          rows={3}
          value={userData}
          onChange={(e) => setUserData(e.target.value)}
          placeholder="#!/bin/sh"
          data-testid="input-userdata"
        />
      </label>
      {fits === false && (
        <p className="text-sm text-warn" data-testid="capacity-warning">
          Selected type does not fit the current schedulable capacity; creation may be rejected.
        </p>
      )}
      <label className="flex items-start gap-2 text-sm" data-testid="trust-ack">
        <input type="checkbox" checked={trusted} onChange={(e) => setTrusted(e.target.checked)} className="mt-0.5" required />
        <span>
          I understand Orbit instances are containers sharing my host kernel — not a secure
          multi-tenant hypervisor.
        </span>
      </label>
      {error && <p className="text-sm text-bad" data-testid="create-error">{error}</p>}
      <button type="submit" disabled={busy || !trusted || !effectiveKey} className="btn-primary" data-testid="create-submit">
        {busy ? "Creating…" : "Create"}
      </button>
    </form>
  );
}
