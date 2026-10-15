// The JS API layer: the only file that knows URLs. Components call these functions.
async function request(path, options = {}) {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`);
  return data;
}

const post = (path, body) => request(path, { method: "POST", body: body ? JSON.stringify(body) : undefined });

export const api = {
  disputes: () => request("/api/disputes"),
  analyze: (id) => post(`/api/disputes/${id}/analyze`),
  approve: (id, response) => post(`/api/disputes/${id}/approve`, { response }),
  reject: (id) => post(`/api/disputes/${id}/reject`),
  policy: () => request("/api/policy"),
  setPolicy: (p) => request("/api/policy", { method: "PUT", body: JSON.stringify(p) }),
  audit: () => request("/api/audit"),
};
