import { useEffect, useState } from "react";
import { motion } from "motion/react";

export const money = (n, c = "USD") => new Intl.NumberFormat("en-US", { style: "currency", currency: c }).format(n);

// Re-render every 30s so countdowns stay live.
export function useNow() {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 30000);
    return () => clearInterval(t);
  }, []);
  return now;
}

// Deadline ring: PayPal gives sellers a 10-day response window; the ring drains as it closes.
export function DeadlineRing({ dueAt, now, size = 54 }) {
  const msLeft = new Date(dueAt).getTime() - now;
  const days = Math.max(0, msLeft / 86400000);
  const frac = Math.min(1, days / 10);
  const color = days < 3 ? "var(--color-rose)" : days < 6 ? "var(--color-amber)" : "var(--color-mint)";
  const r = size / 2 - 4, c = 2 * Math.PI * r;
  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--color-line)" strokeWidth="4" />
        <motion.circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke={color} strokeWidth="4" strokeLinecap="round"
          strokeDasharray={c} initial={{ strokeDashoffset: c }} animate={{ strokeDashoffset: c * (1 - frac) }} transition={{ duration: 0.9 }} />
      </svg>
      <div className="absolute inset-0 grid place-items-center font-mono text-xs" style={{ color }}>
        {days >= 1 ? `${Math.floor(days)}d` : `${Math.max(0, Math.floor(msLeft / 3600000))}h`}
      </div>
    </div>
  );
}

const CHIP = {
  new: ["NEW", "text-zinc-300 border-zinc-600"],
  awaiting_approval: ["AWAITING YOU", "text-amber border-amber/50"],
  needs_human: ["NEEDS A HUMAN", "text-rose border-rose/50"],
  resolved: ["RESOLVED", "text-mint border-mint/50"],
  rejected: ["HANDLED MANUALLY", "text-zinc-400 border-zinc-600"],
};
export function StatusChip({ status }) {
  const [label, cls] = CHIP[status] || CHIP.new;
  return <span className={`rounded-full border px-2 py-0.5 font-mono text-[10px] tracking-wider ${cls}`}>{label}</span>;
}

// Typewriter: the agent's reasoning "types out" so the demo shows it thinking.
export function Typewriter({ text, speed = 12 }) {
  const [n, setN] = useState(0);
  useEffect(() => {
    setN(0);
    const t = setInterval(() => setN((v) => (v >= text.length ? (clearInterval(t), v) : v + 2)), speed);
    return () => clearInterval(t);
  }, [text, speed]);
  return <span>{text.slice(0, n)}<span className="opacity-40">{n < text.length ? "▍" : ""}</span></span>;
}

export function PolicyPanel({ policy, onSave }) {
  const [p, setP] = useState(policy);
  useEffect(() => setP(policy), [policy]);
  if (!p) return null;
  const field = (key, label, step = 1) => (
    <label className="flex items-center justify-between gap-3 text-sm text-zinc-400">
      {label}
      <input type="number" step={step} value={p[key]} onChange={(e) => setP({ ...p, [key]: Number(e.target.value) })}
        className="w-24 rounded-md border border-line bg-ink px-2 py-1 text-right font-mono text-zinc-100 outline-none focus:border-amber" />
    </label>
  );
  return (
    <div className="space-y-3">
      {field("auto_refund_below", "Auto-refund at or below ($)")}
      {field("escalate_at_or_above", "Always ask me at or above ($)")}
      {field("min_confidence", "Min AI confidence (0–1)", 0.05)}
      <button onClick={() => onSave(p)} className="w-full rounded-md border border-line py-1.5 text-sm hover:border-amber hover:text-amber">Save policy</button>
    </div>
  );
}
