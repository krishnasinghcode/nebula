import { useEffect, useRef, useState, type ReactNode } from "react";

// Section-level "i" button: click opens a popover with a one-line brief;
// a "See more" toggle reveals the detailed bullets. Closes on outside
// click or Escape.
export function Info({ brief, children, label = "What does this section do?" }: { brief: string; children?: ReactNode; label?: string }) {
  const [open, setOpen] = useState(false);
  const [more, setMore] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!open) return;
    function onDoc(e: MouseEvent) {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <span className="relative ml-1 inline-flex" ref={ref}>
      <button
        type="button"
        aria-label={label}
        aria-expanded={open}
        onClick={(e) => {
          e.preventDefault();
          setOpen((v) => !v);
          setMore(false);
        }}
        className={`flex h-4 w-4 cursor-pointer items-center justify-center rounded-full border text-[10px] font-semibold leading-none ${
          open
            ? "border-blue-400 text-blue-400"
            : "border-zinc-700 text-zinc-600 hover:border-zinc-500 hover:text-zinc-400"
        }`}
        data-testid="info-button"
      >
        i
      </button>
      {open && (
        <div
          className="absolute left-1/2 top-full z-50 mt-2 w-80 -translate-x-1/2 rounded-lg border border-line bg-panel p-3 text-sm normal-case leading-relaxed text-ink shadow-xl"
          data-testid="info-popover"
        >
          <p>{brief}</p>
          {children && (
            <>
              <button
                type="button"
                className="mt-1.5 text-xs font-medium text-blue-400 hover:text-blue-300"
                onClick={() => setMore((v) => !v)}
                data-testid="info-more"
              >
                {more ? "See less ▴" : "See more ▾"}
              </button>
              {more && <div className="mt-2 border-t border-zinc-800 pt-2">{children}</div>}
            </>
          )}
        </div>
      )}
    </span>
  );
}

// Standard bullet list for Info popovers.
export function InfoList({ items }: { items: [string, ReactNode][] }) {
  return (
    <ul className="list-disc space-y-1.5 pl-4">
      {items.map(([k, v]) => (
        <li key={k}>
          <span className="font-medium">{k}</span> — {v}
        </li>
      ))}
    </ul>
  );
}
