import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CreateWizard } from "./components/CreateWizard";
import type { Instance } from "./api/client";
import { InstanceTable } from "./components/InstanceTable";

const images = [{ id: "ubuntu-24.04", name: "Ubuntu 24.04 LTS", os: "ubuntu", description: "", size_estimate: "" }];
const types = [{ id: "orbit.micro", vcpu: 1, memory_mb: 1024, description: "1 vCPU / 1 GB" }];
const capacity = {
  host: { total_cpus: 4, total_memory_mb: 8192, schedulable_cpus: 3, schedulable_memory_mb: 6144 },
  used_cpus: 0,
  used_memory_mb: 0,
  instances: 0,
  max_instances: 5,
  quota_memory_mb: 8192,
  used_volume_mb: 0,
  quota_volume_mb: 20480,
  subnet: "172.28.0.0/16",
};

afterEach(() => vi.unstubAllGlobals());

describe("CreateWizard", () => {
  it("requires the trust-model acknowledgement before enabling submit", async () => {
    render(<CreateWizard images={images} types={types} keys={["dev"]} capacity={capacity} onCreated={() => {}} />);
    const submit = screen.getByTestId("create-submit") as HTMLButtonElement;
    expect(submit).toBeDisabled();
    await userEvent.click(screen.getByTestId("trust-ack"));
    expect(screen.getByRole("checkbox")).toBeChecked();
  });

  it("submits a create request and calls onCreated", async () => {
    const created = vi.fn();
    const post = vi.fn().mockResolvedValue({
      ok: true,
      status: 202,
      json: async () => ({ id: "orb-1", name: "web-1", state: "provisioning" }),
    });
    vi.stubGlobal("fetch", post);

    render(<CreateWizard images={images} types={types} keys={["dev"]} capacity={capacity} onCreated={created} />);
    await userEvent.type(screen.getByTestId("input-name"), "web-1");
    await userEvent.selectOptions(screen.getByTestId("input-image"), "ubuntu-24.04");
    await userEvent.selectOptions(screen.getByTestId("input-type"), "orbit.micro");
    await userEvent.click(screen.getByTestId("trust-ack"));
    await userEvent.click(screen.getByTestId("create-submit"));

    await waitFor(() => expect(created).toHaveBeenCalled());
    const [, init] = post.mock.calls[0];
    const body = JSON.parse(init.body);
    expect(body).toMatchObject({ name: "web-1", image_id: "ubuntu-24.04", type_id: "orbit.micro", key_id: "dev" });
  });

  it("surfaces API errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        text: async () => JSON.stringify({ error: { code: "validation_failed", message: "insufficient capacity" } }),
      }),
    );
    render(<CreateWizard images={images} types={types} keys={["dev"]} capacity={capacity} onCreated={() => {}} />);
    await userEvent.type(screen.getByTestId("input-name"), "web-1");
    await userEvent.selectOptions(screen.getByTestId("input-image"), "ubuntu-24.04");
    await userEvent.selectOptions(screen.getByTestId("input-type"), "orbit.micro");
    await userEvent.click(screen.getByTestId("trust-ack"));
    await userEvent.click(screen.getByTestId("create-submit"));
    await waitFor(() => expect(screen.getByTestId("create-error")).toHaveTextContent("insufficient capacity"));
  });
});

const rows: Instance[] = [
  { id: "orb-1", name: "web-1", state: "running", image_id: "ubuntu-24.04", type_id: "orbit.micro" },
  { id: "orb-2", name: "db-1", state: "stopped", image_id: "debian-12", type_id: "orbit.small" },
];

describe("InstanceTable", () => {
  it("renders one row per instance with its state", () => {
    render(<InstanceTable instances={rows} onSelect={() => {}} />);
    expect(screen.getByTestId("row-web-1")).toHaveTextContent("running");
    expect(screen.getByTestId("row-db-1")).toHaveTextContent("stopped");
  });

  it("calls onSelect when a row is clicked", async () => {
    const onSelect = vi.fn();
    render(<InstanceTable instances={rows} onSelect={onSelect} />);
    await userEvent.click(screen.getByTestId("row-web-1"));
    expect(onSelect).toHaveBeenCalledWith("orb-1");
  });
});
