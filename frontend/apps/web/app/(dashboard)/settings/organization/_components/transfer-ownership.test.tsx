/** @vitest-environment jsdom */
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { transferOwnership } from "@/actions/organization";

import {
  ana,
  bruno,
  carla,
  renderWithMessages,
  stubBrowserApis,
} from "../../members/_components/test-utils";

import { TransferOwnership } from "./transfer-ownership";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/organization", () => ({ transferOwnership: vi.fn() }));

const transferOwnershipMock = vi.mocked(transferOwnership);

beforeEach(() => {
  stubBrowserApis();
  transferOwnershipMock.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

async function confirmTransferToCarla(
  user: ReturnType<typeof renderWithMessages>
) {
  await user.click(
    screen.getByRole("option", { name: "Carla Viewer (Viewer)" })
  );
  await user.click(screen.getByRole("button", { name: "Transferir" }));
  const dialog = screen.getByRole("alertdialog", {
    name: "Transferir a propriedade para Carla Viewer?",
  });
  await user.click(within(dialog).getByRole("button", { name: "Transferir" }));
}

describe("TransferOwnership", () => {
  it("offers every other Member and transfers to the chosen one", async () => {
    transferOwnershipMock.mockResolvedValue({ kind: "ok" });
    const user = renderWithMessages(
      <TransferOwnership
        policy={{ kind: "canTransfer", candidates: [bruno, carla] }}
      />
    );

    expect(
      screen
        .getByRole("button", { name: "Transferir" })
        .hasAttribute("disabled")
    ).toBe(true);
    await user.click(screen.getByRole("combobox", { name: "Novo Owner" }));
    expect(
      screen.getAllByRole("option").map((option) => option.textContent)
    ).toEqual(["Bruno Admin (Admin)", "Carla Viewer (Viewer)"]);
    await confirmTransferToCarla(user);

    expect(transferOwnershipMock).toHaveBeenCalledWith({ memberId: "m-carla" });
    expect(
      await screen.findByText("Propriedade transferida para Carla Viewer.")
    ).toBeTruthy();
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("explains a refused transfer and keeps the page", async () => {
    transferOwnershipMock.mockResolvedValue({
      kind: "error",
      code: "forbidden",
    });
    const user = renderWithMessages(
      <TransferOwnership
        policy={{ kind: "canTransfer", candidates: [carla] }}
      />
    );

    await user.click(screen.getByRole("combobox", { name: "Novo Owner" }));
    await confirmTransferToCarla(user);

    expect(
      await screen.findByText("Só o Owner pode transferir a propriedade.")
    ).toBeTruthy();
    expect(refresh).not.toHaveBeenCalled();
  });

  it("says when nobody can receive the ownership", () => {
    renderWithMessages(
      <TransferOwnership policy={{ kind: "canTransfer", candidates: [] }} />
    );

    expect(
      screen.getByText("Não há outro Member para receber a propriedade.")
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Transferir" })).toBeNull();
  });

  it("names the Owner to the other Members", () => {
    renderWithMessages(
      <TransferOwnership policy={{ kind: "ownedBy", owner: ana }} />
    );

    expect(
      screen.getByText(
        "Ana Owner é o Owner desta Organization. Só o Owner pode transferir a propriedade."
      )
    ).toBeTruthy();
    expect(screen.queryByRole("combobox")).toBeNull();
  });
});
