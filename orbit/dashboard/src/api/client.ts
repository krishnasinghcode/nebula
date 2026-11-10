import type { components } from "./schema";

export type Instance = components["schemas"]["Instance"];
export type Port = components["schemas"]["Port"];
export type CreateInstanceRequest = components["schemas"]["CreateInstanceRequest"];
export type Image = components["schemas"]["Image"];
export type InstanceType = components["schemas"]["InstanceType"];
export type SSHKey = components["schemas"]["SSHKey"];
export type Capacity = components["schemas"]["Capacity"];
export type Event = components["schemas"]["Event"];

export class ApiError extends Error {
  constructor(
    public status: number,
    body: unknown,
  ) {
    const e = body as { error?: { message?: string } };
    super(e?.error?.message ?? `API error ${status}`);
  }
}

let token = localStorage.getItem("orbit-token") ?? "";

export function setToken(t: string) {
  token = t.trim();
  localStorage.setItem("orbit-token", token);
}

export function hasToken() {
  return token.length >= 32;
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    let parsed: unknown = text;
    try {
      parsed = JSON.parse(text);
    } catch {
      // keep raw text
    }
    throw new ApiError(res.status, parsed);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  listInstances: () => req<Instance[]>("GET", "/v1/instances"),
  getInstance: (id: string) => req<Instance>("GET", `/v1/instances/${id}`),
  getInstanceEvents: (id: string) => req<Event[]>("GET", `/v1/instances/${id}/events`),
  createInstance: (r: CreateInstanceRequest) => req<Instance>("POST", "/v1/instances", r),
  action: (id: string, action: "start" | "stop" | "reboot" | "terminate") =>
    req<unknown>("POST", `/v1/instances/${id}/${action}`),
  updatePorts: (id: string, ports: Port[]) => req<Instance>("PUT", `/v1/instances/${id}/ports`, { ports }),
  listImages: () => req<Image[]>("GET", "/v1/images"),
  listTypes: () => req<InstanceType[]>("GET", "/v1/instance-types"),
  capacity: () => req<Capacity>("GET", "/v1/capacity"),
  listKeys: () => req<SSHKey[]>("GET", "/v1/keys"),
  createKey: (name: string, publicKey: string) =>
    req<SSHKey>("POST", "/v1/keys", { name, public_key: publicKey }),
  deleteKey: (name: string) => req<unknown>("DELETE", `/v1/keys/${name}`),
};
