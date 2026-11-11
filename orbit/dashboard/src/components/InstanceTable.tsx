import { useState } from "react";
import type { Instance } from "../api/client";
import { Info, InfoList } from "./Info";
import { StateBadge } from "./StateBadge";

const FILTERS = ["active", "running", "stopped", "provisioning", "failed", "terminated"] as const;
type Filter = (typeof FILTERS)[number];

const PAGE_SIZE = 5;

// Table extracted for testability; App renders it inline.
export function InstanceTable({
  instances,
  onSelect,
}: {
  instances: Instance[];
  onSelect: (id: string) => void;
}) {
  const [filter, setFilter] = useState<Filter>("active");
  const [page, setPage] = useState(0);

  const visible = instances.filter((i) =>
    filter === "active" ? i.state !== "terminated" : i.state === filter,
  );
  const pages = Math.max(1, Math.ceil(visible.length / PAGE_SIZE));
  const cur = Math.min(page, pages - 1);
  const slice = visible.slice(cur * PAGE_SIZE, cur * PAGE_SIZE + PAGE_SIZE);

  function pick(f: Filter) {
    setFilter(f);
    setPage(0);
  }

  return (
    <section className="rounded-lg border border-line bg-panel" data-testid="instance-table">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line px-3 py-2">
        <h2 className="flex items-center text-sm font-semibold uppercase tracking-wider">
          Instances
          <Info brief="The instance list for this control plane.">
            <InfoList
              items={[
                ["Rows", <>every instance on this control plane — the cloud-style equivalent of the EC2 instance list. Click a row for details, lifecycle actions and ports.</>],
                ["Filters", <>chips above the table show one state only; <b>active</b> (default) is everything except terminated.</>],
                ["Pagination", <>5 rows per page; use Prev/Next below the table.</>],
              ]}
            />
          </Info>
        </h2>
        <div className="flex flex-wrap gap-1" data-testid="state-filters">
          {FILTERS.map((f) => (
            <button
              key={f}
              onClick={() => pick(f)}
              data-testid={`filter-${f}`}
              title={
                f === "active"
                  ? "Everything except terminated instances (the default view)"
                  : `Show only ${f} instances`
              }
              className={`rounded px-2 py-0.5 font-mono text-[11px] transition-colors ${
                filter === f
                  ? "bg-brand text-brand-ink"
                  : "border border-line text-dim hover:text-ink"
              }`}
            >
              {f}
            </button>
          ))}
        </div>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-line text-left">
            <th className="th-label">Name</th>
            <th className="th-label">State</th>
            <th className="th-label">Image</th>
            <th className="th-label">Type</th>
            <th className="th-label">Private IP</th>
            <th className="th-label">Ports</th>
          </tr>
        </thead>
        <tbody>
          {slice.map((i) => (
            <tr
              key={i.id}
              className="cursor-pointer border-b border-line transition-colors last:border-0 hover:bg-hover"
              onClick={() => onSelect(i.id!)}
              data-testid={`row-${i.name}`}
            >
              <td className="px-3 py-2 font-medium">{i.name}</td>
              <td className="px-3 py-2"><StateBadge state={i.state ?? ""} /></td>
              <td className="px-3 py-2 text-dim">{i.image_id}</td>
              <td className="px-3 py-2 font-mono text-xs text-dim">{i.type_id}</td>
              <td className="px-3 py-2 font-mono text-xs text-dim">{i.private_ip ?? "—"}</td>
              <td className="px-3 py-2 font-mono text-xs text-dim">
                {(i.ports ?? []).map((p) => `${p.instance_port}→${p.host_port === 0 ? "?" : p.host_port}`).join(", ") || "—"}
              </td>
            </tr>
          ))}
          {slice.length === 0 && (
            <tr>
              <td className="p-6 text-center text-dim" colSpan={6}>
                No instances. Create one below.
              </td>
            </tr>
          )}
        </tbody>
      </table>
      {pages > 1 && (
        <div className="flex items-center justify-end gap-3 border-t border-line px-3 py-2 text-xs text-dim">
          <button className="rounded border border-line px-2 py-1 disabled:opacity-40" disabled={cur === 0} onClick={() => setPage(cur - 1)} data-testid="page-prev">
            ← Prev
          </button>
          <span className="font-mono" data-testid="page-indicator">
            {cur + 1} / {pages}
          </span>
          <button className="rounded border border-line px-2 py-1 disabled:opacity-40" disabled={cur >= pages - 1} onClick={() => setPage(cur + 1)} data-testid="page-next">
            Next →
          </button>
        </div>
      )}
    </section>
  );
}
