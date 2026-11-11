import { useState } from "react";
import { api, type Instance, type Port } from "../api/client";
import { Info, InfoList } from "./Info";
import { StateBadge } from "./StateBadge";
import { toast } from "./Toast";

export function InstanceDetail({ instance, onBack }: { instance: Instance; onBack: () => void }) {
  const [ports, setPorts] = useState<Port[]>(instance.ports ?? []);
  const editable = instance.state === "stopped";

  const id = instance.id!;

  async function act(action: "start" | "stop" | "reboot" | "terminate") {
    try {
      await api.action(id, action);
      toast(`${instance.name}: ${action} requested`, "success");
    } catch (e) {
      toast(e instanceof Error ? e.message : String(e));
    }
  }

  async function savePorts() {
    try {
      await api.updatePorts(id, ports.map((p) => ({ ...p, host_port: p.host_port || 0 })));
      toast("Ports saved — start the instance to apply.", "success");
    } catch (e) {
      toast(e instanceof Error ? e.message : String(e));
    }
  }

  const sshPort = instance.ports?.find((p) => p.instance_port === 22)?.host_port;

  return (
    <div className="space-y-4" data-testid="instance-detail">
      <button className="text-sm text-dim hover:text-ink" onClick={onBack}>
        ← All instances
      </button>
      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold">{instance.name}</h2>
        <StateBadge state={instance.state ?? ""} />
        <code className="font-mono text-xs text-dim">{instance.id}</code>
      </div>
      {instance.transition_error && <p className="text-sm text-bad">{instance.transition_error}</p>}
      <dl className="grid grid-cols-2 gap-x-4 gap-y-2 rounded-lg border border-line bg-panel p-4 text-sm md:grid-cols-4">
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-dim">Image</dt>
          <dd className="text-dim">{instance.image_id}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-dim">Type</dt>
          <dd className="font-mono text-xs">{instance.type_id}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-dim">Private IP</dt>
          <dd className="font-mono text-xs">{instance.private_ip ?? "—"}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-dim">Key</dt>
          <dd className="font-mono text-xs">{instance.key_id}</dd>
        </div>
      </dl>
      {instance.state === "running" && sshPort ? (
        <p className="text-sm">
          SSH: <code className="rounded border border-line bg-panel px-2 py-1 font-mono text-xs" data-testid="ssh-command">orbit ssh {instance.name}</code>{" "}
          <span className="font-mono text-xs text-dim">(host port {sshPort})</span>
        </p>
      ) : null}
      <div className="flex gap-2">
        <button className="btn-primary disabled:opacity-40" disabled={instance.state !== "stopped"} onClick={() => act("start")} title="Power the instance on" data-testid="btn-start">
          Start
        </button>
        <button className="rounded border border-line px-3 py-1.5 text-sm font-medium transition-colors hover:bg-hover disabled:opacity-40" disabled={instance.state !== "running"} onClick={() => act("stop")} title="Shut the instance down (state kept; ports editable while stopped)" data-testid="btn-stop">
          Stop
        </button>
        <button className="rounded border border-line px-3 py-1.5 text-sm font-medium transition-colors hover:bg-hover disabled:opacity-40" disabled={instance.state !== "running"} onClick={() => act("reboot")} title="Restart the instance">
          Reboot
        </button>
        <button className="btn-danger" onClick={() => { if (confirm(`Terminate ${instance.name}? Irreversible.`)) act("terminate"); }} title="Destroy the instance permanently; its name becomes reusable" data-testid="btn-terminate">
          Terminate
        </button>
      </div>
      <section className="rounded-lg border border-line bg-panel p-4">
        <h3 className="mb-2 flex items-center text-sm font-semibold uppercase tracking-wider">
          Published ports
          <Info brief="Host ports forwarded into the instance.">
            <InfoList
              items={[
                ["SSH", <>port 22 is published automatically (loopback only) — that's the 22→port shown in the list.</>],
                ["Guest port", <>the port your app listens on inside the instance.</>],
                ["Host port", <>the port on your machine; 0 = auto-assign a free one.</>],
                ["Editing", <>only while the instance is stopped; changes apply on start.</>],
              ]}
            />
          </Info>
        </h3>
        {ports.length === 0 && <p className="text-sm text-dim">None (SSH is published automatically).</p>}
        <ul className="space-y-1">
          {ports.map((p, i) => (
            <li key={i} className="flex items-center gap-2 font-mono text-xs">
              <span className="text-dim">
                {p.instance_port} → {p.host_port === 0 ? "auto" : p.host_port} ({p.bind ?? "loopback"})
              </span>
              {editable && (
                <button className="text-xs text-bad hover:opacity-75" onClick={() => setPorts(ports.filter((_, j) => j !== i))}>
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
            <input name="ip" type="number" min={1} max={65535} placeholder="guest port" required className="console-input w-32 font-mono text-xs" />
            <input name="hp" type="number" min={0} max={65535} placeholder="host port (0=auto)" className="console-input w-40 font-mono text-xs" />
            <button className="rounded border border-line px-2 py-1 text-sm transition-colors hover:bg-hover">Add</button>
            <button type="button" onClick={savePorts} className="btn-primary px-2 py-1 text-xs" data-testid="save-ports">
              Save
            </button>
          </form>
        )}
        {!editable && ports.length > 0 && (
          <p className="mt-2 text-xs text-dim">Editing available while the instance is stopped.</p>
        )}
      </section>
    </div>
  );
}
