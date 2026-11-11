import { useEffect, useState } from "react";

// Tiny global toast bus: call toast(msg) / toast(msg, "success") from
// anywhere; <Toasts/> renders the stack (bottom-right, auto-dismiss).
type Item = { id: number; msg: string; kind: "error" | "success" };

let push: ((msg: string, kind: "error" | "success") => void) | null = null;
let nextId = 1;

export function toast(msg: string, kind: "error" | "success" = "error") {
  push?.(msg, kind);
}

export function Toasts() {
  const [items, setItems] = useState<Item[]>([]);

  useEffect(() => {
    push = (msg, kind) => {
      const id = nextId++;
      setItems((cur) => [...cur, { id, msg, kind }]);
      setTimeout(() => setItems((cur) => cur.filter((t) => t.id !== id)), 4500);
    };
    return () => {
      push = null;
    };
  }, []);

  const cls =
    "rounded-md border px-3 py-2 text-sm shadow-lg backdrop-blur max-w-sm";
  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2" data-testid="toasts">
      {items.map((t) => (
        <div
          key={t.id}
          data-testid={`toast-${t.kind}`}
          role="status"
          className={
            t.kind === "error"
              ? `${cls} border-bad bg-panel text-bad`
              : `${cls} border-ok bg-panel text-ok`
          }
        >
          {t.msg}
        </div>
      ))}
    </div>
  );
}
