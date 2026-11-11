// State dot + mono label — the console-style status indicator.
const dots: Record<string, string> = {
  running: "bg-ok",
  provisioning: "bg-brand",
  rebooting: "bg-brand",
  stopping: "bg-warn",
  terminating: "bg-warn",
  stopped: "bg-dim",
  failed: "bg-bad",
  terminated: "bg-transparent border border-dim",
};

export function StateBadge({ state }: { state: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`inline-block h-2 w-2 rounded-full ${dots[state] ?? "bg-dim"}`} />
      <span className="font-mono text-xs">{state}</span>
    </span>
  );
}
