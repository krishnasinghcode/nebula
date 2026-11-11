import { useState } from "react";
import { api, hasToken, setToken, type Image, type InstanceType } from "./api/client";
import { useCapacity, useInstances, usePoll } from "./hooks/usePoll";
import { CreateWizard } from "./components/CreateWizard";
import { Info } from "./components/Info";
import { InstanceDetail } from "./components/InstanceDetail";
import { InstanceTable } from "./components/InstanceTable";
import { KeysPanel } from "./components/KeysPanel";
import { Toasts } from "./components/Toast";

function ThemeToggle() {
  const [theme, setTheme] = useState(document.documentElement.dataset.theme ?? "light");
  function flip() {
    const next = theme === "light" ? "dark" : "light";
    document.documentElement.dataset.theme = next;
    localStorage.setItem("orbit-theme", next);
    setTheme(next);
  }
  return (
    <button
      onClick={flip}
      className="rounded border border-line px-2 py-1 text-xs text-dim hover:text-ink"
      title={theme === "light" ? "Switch to dark theme" : "Switch to light theme"}
      data-testid="theme-toggle"
    >
      {theme === "light" ? "☾ Dark" : "☀ Light"}
    </button>
  );
}

// Thin segmented usage meter (vCPU / memory / instances).
function Meter({ label, used, max, unit }: { label: string; used: number; max: number; unit: string }) {
  const pct = max > 0 ? Math.min(100, Math.round((used / max) * 100)) : 0;
  const near = pct >= 90;
  return (
    <span className="flex items-center gap-1.5" title={`${used}/${max} ${unit}`}>
      <span className="text-[10px] uppercase tracking-wider text-dim">{label}</span>
      <span className="flex h-1.5 w-16 gap-px overflow-hidden rounded-full bg-line">
        <span className={`h-full ${near ? "bg-bad" : "bg-brand"}`} style={{ width: `${pct}%` }} />
      </span>
      <span className="font-mono text-[11px] text-dim">
        {used}/{max}
      </span>
    </span>
  );
}

function TokenGate({ children }: { children: React.ReactNode }) {
  const [value, setValue] = useState("");
  if (hasToken()) return <>{children}</>;
  return (
    <div className="mx-auto mt-24 max-w-md space-y-3 rounded-lg border border-line bg-panel p-6 shadow-sm" data-testid="token-gate">
      <h1 className="font-semibold">Connect to Orbit</h1>
      <p className="text-sm text-dim">
        Paste your API token. Find it at <code className="font-mono text-ink">~/.orbit/token</code> on the machine running{" "}
        <code className="font-mono text-ink">orbit serve</code>.
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          setToken(value);
        }}
        className="flex gap-2"
      >
        <input
          className="console-input flex-1"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="token"
          type="password"
          required
        />
        <button className="btn-primary">Connect</button>
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
    if (!confirm("Clear the stored token? You'll need to paste the token again to reconnect.")) return;
    // Re-run the connect gate: clears the stored token (backend switch/re-auth).
    localStorage.removeItem("orbit-token");
    location.reload();
  }

  return (
    <>
      <Toasts />
      {/* Product shell: full-width console bar */}
      <header className="sticky top-0 z-40 border-b border-line bg-panel">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 px-6 py-2.5">
          <h1 className="text-sm font-semibold uppercase tracking-widest">
            Nebula <span className="text-brand">Orbit</span>
          </h1>
          <div className="flex flex-wrap items-center gap-3">
            {capacity && (
              <span className="flex items-center gap-3" data-testid="capacity-summary">
                <Info brief="Resource usage vs the schedulable limits of this machine.">
                  <span className="block w-56 text-xs text-dim">
                    What new instances can still consume on this host.
                  </span>
                </Info>
                <Meter label="vCPU" used={capacity.used_cpus ?? 0} max={capacity.host!.schedulable_cpus ?? 0} unit="vCPU" />
                <Meter label="Mem" used={capacity.used_memory_mb ?? 0} max={capacity.host!.schedulable_memory_mb ?? 0} unit="MB" />
                <Meter label="Inst" used={capacity.instances ?? 0} max={capacity.max_instances ?? 0} unit="instances" />
              </span>
            )}
            <ThemeToggle />
            <button
              className="text-xs text-dim underline hover:text-ink"
              onClick={changeToken}
              data-testid="change-token"
              title="Clear the stored API token and reconnect (e.g. after switching backends)"
            >
              Change token
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl space-y-6 p-6">
        {instancesError?.message.includes("401") && (
          <p className="rounded border border-bad bg-bad/10 p-3 text-sm" style={{ color: "var(--c-bad)" }} data-testid="auth-error">
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
    </>
  );
}

export default function App() {
  return (
    <TokenGate>
      <Dashboard />
    </TokenGate>
  );
}
