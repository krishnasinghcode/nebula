import { useState } from "react";
import { api, type Instance, type Port } from "../api/client";
import { StateBadge } from "./StateBadge";

export function InstanceDetail({ instance, onBack }: { instance: Instance; onBack: () => void }) {
  const [ports, setPorts] = useState<Port[]>(instance.ports ?? []);
  const [msg, setMsg] = useState<string | null>(null);
  const editable = instance.state === "stopped";

  const id = instance.id!;

  async function act(action: "start" | "stop" | "reboot" | "terminate") {
    setMsg(null);
    try {
      await api.action(id, action);
    } catch (e) {
      setMsg(String(e));
    }
  }

  async function savePorts() {
    setMsg(null);
    try {
      await api.updatePorts(id, ports.map((p) => ({ ...p, host_port: p.host_port || 0 })));
      setMsg("Ports saved — start the instance to apply.");
    } catch (e) {
      setMsg(e instanceof Error ? e.message : String(e));
    }
  }

  async function refresh() {
    // replaced by parent polling re-render; kept for clarity of the copy button
  }
  void refresh;

  const sshPort = instance.ports?.find((p) => p.instance_port === 22)?.host_port;

  return (
    <div className="space-y-4" data-testid="instance-detail">
      <button className="text-sm text-zinc-400 hover:text-zinc-200" onClick={onBack}>
        ← All instances
      </button>
      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold">{instance.name}</h2>
        <StateBadge state={instance.state ?? ""} />
        <code className="text-xs text-zinc-500">{instance.id}</code>
      </div>
      {instance.transition_error && <p className="text-sm text-red-400">{instance.transition_error}</p>}
      <dl className="grid grid-cols-2 gap-2 text-sm md:grid-cols-4">
        <div>
          <dt className="text-zinc-500">Image</dt>
          <dd>{instance.image_id}</dd>
        </div>
        <div>
          <dt className="text-zinc-500">Type</dt>
          <dd>{instance.type_id}</dd>
        </div>
        <div>
          <dt className="text-zinc-500">Private IP</dt>
          <dd>{instance.private_ip ?? "—"}</dd>
        </div>
        <div>
          <dt className="text-zinc-500">Key</dt>
          <dd>{instance.key_id}</dd>
        </div>
      </dl>
      {instance.state === "running" && sshPort ? (
        <p className="text-sm">
          SSH: <code className="rounded bg-zinc-900 px-2 py-1" data-testid="ssh-command">orbit ssh {instance.name}</code>{" "}
          <span className="text-zinc-500">(host port {sshPort})</span>
        </p>
      ) : null}
      <div className="flex gap-2">
        <button className="rounded bg-green-700 px-3 py-1.5 text-sm hover:bg-green-600 disabled:opacity-40" disabled={instance.state !== "stopped"} onClick={() => act("start")} data-testid="btn-start">
          Start
        </button>
        <button className="rounded bg-yellow-700 px-3 py-1.5 text-sm hover:bg-yellow-600 disabled:opacity-40" disabled={instance.state !== "running"} onClick={() => act("stop")} data-testid="btn-stop">
          Stop
        </button>
        <button className="rounded bg-blue-700 px-3 py-1.5 text-sm hover:bg-blue-600 disabled:opacity-40" disabled={instance.state !== "running"} onClick={() => act("reboot")}>
          Reboot
        </button>
        <button className="rounded bg-red-800 px-3 py-1.5 text-sm hover:bg-red-700" onClick={() => { if (confirm(`Terminate ${instance.name}? Irreversible.`)) act("terminate"); }} data-testid="btn-terminate">
          Terminate
        </button>
      </div>
      <section className="rounded-lg border border-zinc-800 p-4">
        <h3 className="mb-2 text-sm font-semibold">Published ports</h3>
        {ports.length === 0 && <p className="text-sm text-zinc-500">None (SSH is published automatically).</p>}
        <ul className="space-y-1">
          {ports.map((p, i) => (
            <li key={i} className="flex items-center gap-2 text-sm">
              <span>
                {p.instance_port} → {p.host_port === 0 ? "auto" : p.host_port} ({p.bind ?? "loopback"})
              </span>
              {editable && (
                <button className="text-xs text-red-400" onClick={() => setPorts(ports.filter((_, j) => j !== i))}>
                  remove
                </button>
              )}
            </li>
          ))}
        </ul>
        {editable && (
          <form
            className="mt-2 flex gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              const f = e.currentTarget.elements as typeof e.currentTarget.elements & {
                ip: HTMLInputElement;
                hp: HTMLInputElement;
              };
              setPorts([...ports, { instance_port: Number(f.ip.value), host_port: Number(f.hp.value) || 0, bind: "loopback" }]);
              e.currentTarget.reset();
            }}
          >
            <input name="ip" type="number" min={1} max={65535} placeholder="guest port" required className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 text-sm w-32" />
            <input name="hp" type="number" min={0} max={65535} placeholder="host port (0=auto)" className="rounded border border-zinc-700 bg-zinc-900 px-2 py-1 text-sm w-40" />
            <button className="rounded bg-zinc-700 px-2 py-1 text-sm hover:bg-zinc-600">Add</button>
            <button type="button" onClick={savePorts} className="rounded bg-blue-600 px-2 py-1 text-sm hover:bg-blue-500" data-testid="save-ports">
              Save
            </button>
          </form>
        )}
        {!editable && ports.length > 0 && (
          <p className="mt-2 text-xs text-zinc-500">Editing available while the instance is stopped.</p>
        )}
      </section>
      {msg && <p className="text-sm text-yellow-400" data-testid="detail-msg">{msg}</p>}
    </div>
  );
}
