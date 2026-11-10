const colors: Record<string, string> = {
  running: "bg-green-900 text-green-300",
  provisioning: "bg-blue-900 text-blue-300",
  rebooting: "bg-blue-900 text-blue-300",
  stopping: "bg-yellow-900 text-yellow-300",
  terminating: "bg-yellow-900 text-yellow-300",
  stopped: "bg-zinc-800 text-zinc-300",
  failed: "bg-red-900 text-red-300",
  terminated: "bg-zinc-800 text-zinc-500",
};

export function StateBadge({ state }: { state: string }) {
  return (
    <span className={`rounded px-2 py-0.5 text-xs font-medium ${colors[state] ?? "bg-zinc-800"}`}>
      {state}
    </span>
  );
}
