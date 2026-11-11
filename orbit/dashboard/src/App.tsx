import { useState } from "react";
import { api, hasToken, setToken, type Image, type InstanceType } from "./api/client";
import { useCapacity, useInstances, usePoll } from "./hooks/usePoll";
import { CreateWizard } from "./components/CreateWizard";
import { InstanceDetail } from "./components/InstanceDetail";
import { InstanceTable } from "./components/InstanceTable";
import { KeysPanel } from "./components/KeysPanel";

function TokenGate({ children }: { children: React.ReactNode }) {
  const [value, setValue] = useState("");
  if (hasToken()) return <>{children}</>;
  return (
    <div className="mx-auto mt-24 max-w-md space-y-3 rounded-lg border border-zinc-800 p-6" data-testid="token-gate">
      <h1 className="font-semibold">Connect to Orbit</h1>
      <p className="text-sm text-zinc-400">
        Paste your API token. Find it at <code className="text-zinc-300">~/.orbit/token</code> on the machine running{" "}
        <code className="text-zinc-300">orbit serve</code>.
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          setToken(value);
        }}
        className="flex gap-2"
      >
        <input
          className="flex-1 rounded border border-zinc-700 bg-zinc-900 px-2 py-1.5 text-sm"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="token"
          type="password"
          required
        />
        <button className="rounded bg-blue-600 px-3 py-1.5 text-sm hover:bg-blue-500">Connect</button>
      </form>
    </div>
  );
}

function Dashboard() {
  const { data: instances, error: instancesError, refresh: refreshInstances } = useInstances();
  const { data: capacity } = useCapacity();
  const { data: images } = usePoll(api.listImages, 60000);
  const { data: types } = usePoll(api.listTypes, 60000);
  const { data: keys } = usePoll(api.listKeys, 5000);
  const [selected, setSelected] = useState<string | null>(null);

  const list = instances ?? [];
  const current = selected ? list.find((i) => i.id === selected) : undefined;

  function changeToken() {
    // Re-run the connect gate: clears the stored token (backend switch/re-auth).
    localStorage.removeItem("orbit-token");
    location.reload();
  }

  return (
    <main className="mx-auto max-w-5xl p-6 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">
          Nebula <span className="text-blue-400">Orbit</span>
        </h1>
        <div className="flex items-center gap-4">
          {capacity && (
            <span className="text-xs text-zinc-500" data-testid="capacity-summary">
              {capacity.used_cpus}/{capacity.host!.schedulable_cpus} vCPU · {capacity.used_memory_mb}/
              {capacity.host!.schedulable_memory_mb} MB · {capacity.instances}/{capacity.max_instances} instances
            </span>
          )}
          <button
            className="text-xs text-zinc-500 hover:text-zinc-300 underline"
            onClick={changeToken}
            data-testid="change-token"
            title="Clear the stored API token and reconnect (e.g. after switching backends)"
          >
            Change token
          </button>
        </div>
      </header>

      {instancesError?.message.includes("401") && (
        <p className="rounded border border-red-800 bg-red-950 p-3 text-sm text-red-300" data-testid="auth-error">
          The API rejected your token (backend may have been recreated). Use{" "}
          <button className="underline" onClick={changeToken}>Change token</button> and paste the new one.
        </p>
      )}

      {current ? (
        <InstanceDetail instance={current} onBack={() => setSelected(null)} />
      ) : (
        <>
          <InstanceTable instances={list} onSelect={setSelected} />

          <CreateWizard
            images={images ?? ([] as Image[])}
            types={types ?? ([] as InstanceType[])}
            keys={(keys ?? []).map((k) => k.name!).filter(Boolean)}
            capacity={capacity}
            onCreated={refreshInstances}
          />

          <KeysPanel />
        </>
      )}
    </main>
  );
}

export default function App() {
  return (
    <TokenGate>
      <Dashboard />
    </TokenGate>
  );
}
