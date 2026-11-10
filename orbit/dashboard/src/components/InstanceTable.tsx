import type { Instance } from "../api/client";

// Table extracted for testability; App renders it inline.
export function InstanceTable({
  instances,
  onSelect,
}: {
  instances: Instance[];
  onSelect: (id: string) => void;
}) {
  return (
    <section className="rounded-lg border border-zinc-800" data-testid="instance-table">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 text-left text-zinc-500">
            <th className="p-2">Name</th>
            <th className="p-2">State</th>
            <th className="p-2">Image</th>
            <th className="p-2">Type</th>
            <th className="p-2">Private IP</th>
            <th className="p-2">Ports</th>
          </tr>
        </thead>
        <tbody>
          {instances.map((i) => (
            <tr
              key={i.id}
              className="cursor-pointer border-b border-zinc-900 hover:bg-zinc-900"
              onClick={() => onSelect(i.id!)}
              data-testid={`row-${i.name}`}
            >
              <td className="p-2 font-medium">{i.name}</td>
              <td className="p-2">{i.state}</td>
              <td className="p-2">{i.image_id}</td>
              <td className="p-2">{i.type_id}</td>
              <td className="p-2">{i.private_ip ?? "—"}</td>
              <td className="p-2 text-zinc-400">
                {(i.ports ?? []).map((p) => `${p.instance_port}→${p.host_port === 0 ? "?" : p.host_port}`).join(", ") || "—"}
              </td>
            </tr>
          ))}
          {instances.length === 0 && (
            <tr>
              <td className="p-4 text-center text-zinc-500" colSpan={6}>
                No instances yet.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </section>
  );
}
