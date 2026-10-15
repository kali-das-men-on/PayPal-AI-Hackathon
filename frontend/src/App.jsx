import { useCallback, useEffect, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { api } from "./api";
import { DeadlineRing, PolicyPanel, StatusChip, Typewriter, money, useNow } from "./components";

const Card = ({ title, children, className = "" }) => (
  <section className={`rounded-2xl border border-line bg-panel/80 p-4 backdrop-blur ${className}`}>
    {title && <h2 className="mb-3 font-mono text-[11px] uppercase tracking-[0.18em] text-zinc-500">{title}</h2>}
    {children}
  </section>
);

export default function App() {
  const now = useNow();
  const [disputes, setDisputes] = useState([]);
  const [selected, setSelected] = useState(null);
  const [policy, setPolicy] = useState(null);
  const [audit, setAudit] = useState([]);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    const [d, a] = await Promise.all([api.disputes(), api.audit()]);
    setDisputes(d); setAudit(a);
    setSelected((s) => s ?? d[0]?.id ?? null);
  }, []);

  useEffect(() => { refresh().catch((e) => setError(e.message)); api.policy().then(setPolicy).catch(() => {}); }, [refresh]);

  const current = disputes.find((d) => d.id === selected);
  useEffect(() => setDraft(current?.analysis?.draft_response ?? ""), [current?.id, current?.analysis?.draft_response]);

  const act = async (fn) => {
    setBusy(true); setError("");
    try { await fn(); await refresh(); } catch (e) { setError(e.message); } finally { setBusy(false); }
  };

  const open = disputes.filter((d) => !["resolved", "rejected"].includes(d.status));
  const atStake = open.reduce((s, d) => s + d.amount, 0);

  return (
    <div className="mx-auto max-w-6xl px-5 py-8">
      <header className="mb-6 flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dispute<span className="text-amber">Desk</span></h1>
          <p className="text-sm text-zinc-500">An agent that never misses a deadline, and never fights a claim it should settle.</p>
        </div>
        <div className="text-right font-mono text-xs text-zinc-500">
          <div>{open.length} open · {money(atStake)} at stake</div>
          <div className="text-zinc-600">sandbox / demo data only</div>
        </div>
      </header>

      {error && <div className="mb-4 rounded-lg border border-rose/40 bg-rose/10 px-3 py-2 text-sm text-rose">{error}</div>}

      <div className="grid gap-5 lg:grid-cols-[340px_1fr]">
        <div className="space-y-5">
          <Card title="Queue · by deadline">
            <ul className="space-y-2">
              <AnimatePresence initial={false}>
                {disputes.map((d) => (
                  <motion.li key={d.id} layout initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}>
                    <button onClick={() => setSelected(d.id)}
                      className={`flex w-full items-center gap-3 rounded-xl border p-3 text-left transition ${selected === d.id ? "border-amber/60 bg-amber/5" : "border-line hover:border-zinc-600"}`}>
                      <DeadlineRing dueAt={d.due_at} now={now} />
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium">{d.item}</div>
                        <div className="font-mono text-xs text-zinc-500">{d.id} · {money(d.amount, d.currency)}</div>
                        <div className="mt-1"><StatusChip status={d.status} /></div>
                      </div>
                    </button>
                  </motion.li>
                ))}
              </AnimatePresence>
            </ul>
          </Card>
          <Card title="Your policy"><PolicyPanel policy={policy} onSave={(p) => act(async () => setPolicy(await api.setPolicy(p)))} /></Card>
        </div>

        <div className="space-y-5">
          {current ? (
            <Card>
              <div className="mb-4 flex items-start justify-between gap-4">
                <div>
                  <div className="font-mono text-xs text-zinc-500">{current.id} · {current.reason.replaceAll("_", " ").toLowerCase()}</div>
                  <h2 className="text-xl font-semibold">{current.item} <span className="text-zinc-500">· {money(current.amount, current.currency)}</span></h2>
                  <p className="mt-1 text-sm text-zinc-400">Buyer says: “{current.evidence.buyer_message || "—"}”</p>
                </div>
                <DeadlineRing dueAt={current.due_at} now={now} size={64} />
              </div>

              <div className="mb-4 grid grid-cols-3 gap-2 font-mono text-xs">
                {[["Delivery", current.evidence.delivery_status], ["Carrier", current.evidence.carrier || "—"], ["Tracking", current.evidence.tracking_number || "—"]].map(([k, v]) => (
                  <div key={k} className="rounded-lg border border-line p-2"><div className="text-zinc-500">{k}</div><div className="truncate">{v}</div></div>
                ))}
              </div>

              {!current.analysis ? (
                <button disabled={busy} onClick={() => act(() => api.analyze(current.id))}
                  className="w-full rounded-xl bg-amber py-2.5 font-medium text-black transition hover:brightness-110 disabled:opacity-50">
                  {busy ? "Agent is reading the evidence…" : "Run agent on this dispute"}
                </button>
              ) : (
                <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-3">
                  <div className="flex flex-wrap items-center gap-2 text-xs">
                    <span className="rounded-md bg-line px-2 py-1 font-mono">{current.decision.recommendation.toUpperCase()}</span>
                    <span className="font-mono text-zinc-500">evidence: {current.analysis.evidence_strength} · confidence {Math.round(current.analysis.confidence * 100)}% · engine: {current.analysis.engine}</span>
                  </div>
                  <p className="text-sm leading-relaxed text-zinc-300"><Typewriter text={current.analysis.reasoning} /></p>
                  <div className="rounded-lg border border-line p-2 text-xs text-zinc-400">Policy: <span className="text-zinc-200">{current.decision.action.replaceAll("_", " ")}</span>. {current.decision.reason}.</div>
                  <textarea value={draft} onChange={(e) => setDraft(e.target.value)} rows={4} disabled={current.status === "resolved"}
                    className="w-full resize-none rounded-lg border border-line bg-ink p-3 text-sm outline-none focus:border-amber" />
                  {["awaiting_approval", "needs_human"].includes(current.status) && (
                    <div className="flex gap-2">
                      <button disabled={busy} onClick={() => act(() => api.approve(current.id, draft))} className="flex-1 rounded-xl bg-mint py-2 font-medium text-black hover:brightness-110 disabled:opacity-50">Approve & send</button>
                      <button disabled={busy} onClick={() => act(() => api.reject(current.id))} className="rounded-xl border border-line px-4 hover:border-rose hover:text-rose">I'll handle it</button>
                    </div>
                  )}
                </motion.div>
              )}
            </Card>
          ) : <Card><p className="text-sm text-zinc-500">No disputes loaded.</p></Card>}

          <Card title="Audit log · everything the agent saw and did">
            <ul className="max-h-56 space-y-1.5 overflow-auto font-mono text-xs">
              {audit.length === 0 && <li className="text-zinc-600">Nothing yet.</li>}
              {audit.map((e, i) => (
                <li key={i} className="flex gap-3 text-zinc-400"><span className="text-zinc-600">{new Date(e.at).toLocaleTimeString()}</span><span className="text-amber">{e.actor}</span><span>{e.event}</span><span className="text-zinc-600">{e.dispute_id}</span><span className="truncate text-zinc-500">{e.detail}</span></li>
              ))}
            </ul>
          </Card>
        </div>
      </div>
    </div>
  );
}
