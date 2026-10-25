import { afterEach, describe, it, expect, vi } from "vitest";
import {
  api,
  setAdminCredential,
  verifyAdminKey,
  ApiError,
  getPlayers,
  updatePlayer,
  assignCaptain,
  removePlayer,
  getPlayerStats,
  updateFixtureVenue,
} from "./api";
afterEach(() => { setAdminCredential(""); vi.unstubAllGlobals(); });
describe("existing Go API integration contract", () => {
  it("normalizes capitalized Go roster DTOs", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [
              {
                ID: "player-id",
                Name: "Alex",
                JerseyNumber: "10",
                Team: "Team A",
              },
            ],
          }),
        ),
      ),
    );
    expect(await getPlayers("EDL001")).toEqual([
      {
        id: "player-id",
        name: "Alex",
        jersey_number: "10",
        team: "Team A",
        team_tag: "EDL001",
        player_tag: "EDL001-10",
        is_captain: false,
      },
    ]);
  });
  it("sends admin keys only in the request authorization header", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ data: { id: "team" } })),
      );
    vi.stubGlobal("fetch", fetchMock);
    await api(
      "/teams",
      {
        method: "POST",
        body: JSON.stringify({ name: "Team A", tag: "EDL001" }),
      },
      "test-only-key",
    );
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/teams");
    expect(options.headers.get("Authorization")).toBe("Bearer test-only-key");
    expect(options.body).not.toContain("test-only-key");
  });
  it("preserves stale-revision errors for conflict handling", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "Fixture changed; refresh before submitting again",
          }),
          { status: 409 },
        ),
      ),
    );
    try {
      await api("/fixtures/id/result", { method: "PUT" });
      throw new Error("Expected a conflict");
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(409);
    }
  });
  it("handles successful empty responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    );
    expect(await api("/teams/id/players/id", { method: "PUT" })).toEqual({});
  });
});

describe("captains and roster management", () => {
  it("preserves the new DTO captain flag and player tag", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [
              {
                id: "captain-id",
                name: "Alex",
                jersey_number: "10",
                team: "Team A",
                team_tag: "EDL001",
                player_tag: "EDL001-10",
                is_captain: true,
              },
            ],
          }),
        ),
      ),
    );
    const players = await getPlayers("EDL001");
    expect(players[0].is_captain).toBe(true);
    expect(players[0].player_tag).toBe("EDL001-10");
    expect(players[0].team_tag).toBe("EDL001");
  });
  it("updates player details with a numeric jersey and no captain change", async () => {
    const request = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", request);
    await updatePlayer(
      "EDL001",
      "player-id",
      { name: "Alexander", jersey_number: 11 },
      "dummy-admin-key",
    );
    const [url, options] = request.mock.calls[0];
    expect(url).toBe("/api/teams/EDL001/players/player-id");
    expect(options.method).toBe("PUT");
    expect(JSON.parse(options.body)).toEqual({
      name: "Alexander",
      jersey_number: 11,
    });
    expect(options.headers.get("Authorization")).toBe("Bearer dummy-admin-key");
  });
  it("assigns captain through the dedicated authenticated endpoint", async () => {
    const request = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", request);
    await assignCaptain("EDL001", "player-id", "dummy-admin-key");
    const [url, options] = request.mock.calls[0];
    expect(url).toBe("/api/teams/EDL001/players/player-id/captain");
    expect(options.method).toBe("PUT");
    expect(options.headers.get("Authorization")).toBe("Bearer dummy-admin-key");
  });
  it("preserves protected-player deletion errors", async () => {
    const request = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          error: "This player has match records and cannot be removed",
        }),
        { status: 409 },
      ),
    );
    vi.stubGlobal("fetch", request);
    await expect(
      removePlayer("EDL001", "player-id", "dummy-admin-key"),
    ).rejects.toMatchObject({
      status: 409,
      message: "This player has match records and cannot be removed",
    });
    expect(request.mock.calls[0][1].method).toBe("DELETE");
  });
});

describe("simplified statistics and venues", () => {
  it("reads the simplified red_cards field and excludes obsolete statistics", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({
              data: [
                {
                  player_id: "p",
                  name: "Alex",
                  goals: 3,
                  yellow_cards: 2,
                  red_cards: 1,
                  own_goals: 2,
                  dismissals: 99,
                  second_yellow_dismissals: 99,
                },
              ],
            }),
          ),
        ),
    );
    const [player] = await getPlayerStats();
    expect(player.goals).toBe(3);
    expect(player.yellow_cards).toBe(2);
    expect(player.red_cards).toBe(1);
    expect(player).not.toHaveProperty("dismissals");
    expect(player).not.toHaveProperty("second_yellow_dismissals");
    expect(player.own_goals).toBe(2);
  });
  it("supports the existing backend red-card field during the transition", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(JSON.stringify({ data: [{ straight_red_cards: 2 }] })),
        ),
    );
    expect((await getPlayerStats())[0].red_cards).toBe(2);
  });
  it("sends a venue-only PATCH and permits clearing with null", async () => {
    const request = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(new Response(null, { status: 204 })),
      );
    vi.stubGlobal("fetch", request);
    await updateFixtureVenue("fixture-id", "Community Ground", "dummy-key");
    await updateFixtureVenue("fixture-id", null, "dummy-key");
    expect(request.mock.calls[0][0]).toBe("/api/fixtures/fixture-id/venue");
    expect(request.mock.calls[0][1].method).toBe("PATCH");
    expect(JSON.parse(request.mock.calls[0][1].body)).toEqual({
      venue: "Community Ground",
    });
    expect(JSON.parse(request.mock.calls[1][1].body)).toEqual({ venue: null });
    expect(request.mock.calls[0][1].headers.get("Authorization")).toBe(
      "Bearer dummy-key",
    );
  });
});


it("sorts rosters by numeric jersey number with missing numbers last", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({data: [
    {id: "ten", name: "Ten", jersey_number: "10"},
    {id: "missing", name: "Missing", jersey_number: ""},
    {id: "two", name: "Two", jersey_number: "2"},
    {id: "one", name: "One", jersey_number: "1"},
  ]}))));
  expect((await getPlayers("EDL001")).map(p => p.id)).toEqual(["one", "two", "ten", "missing"]);
});

describe("admin login verification", () => {
  it("verifies the key via the protected endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({data:{authenticated:true}})));
    vi.stubGlobal("fetch", fetchMock);
    await verifyAdminKey("secret-test-key");
    expect(fetchMock.mock.calls[0][0]).toBe("/api/admin/session");
    expect(fetchMock.mock.calls[0][1].headers.get("Authorization")).toBe("Bearer secret-test-key");
  });
  it("rejects incorrect keys", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({error:"Invalid admin key"}),{status:401})));
    await expect(verifyAdminKey("wrong")).rejects.toMatchObject({status:401});
  });
  it("does not accept a successful unrelated response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({status:"ready"}))));
    await expect(verifyAdminKey("wrong")).rejects.toThrow("could not be verified");
  });
});

it("uses cookie sessions and the in-memory admin credential for protected reads",async()=>{
 const fetchMock=vi.fn().mockResolvedValue(new Response(JSON.stringify({data:[]})));
 vi.stubGlobal("fetch",fetchMock);setAdminCredential("test-session-key");
 await api("/teams");
 const options=fetchMock.mock.calls[0][1];
 expect(options.credentials).toBe("same-origin");
 expect(options.headers.get("Authorization")).toBe("Bearer test-session-key");
});
it("locks the viewer when the server revokes a session",async()=>{
 const dispatchEvent=vi.fn();vi.stubGlobal("window",{dispatchEvent});
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(new Response(JSON.stringify({error:"Enter PIN",code:"viewer_auth_required"}),{status:401})));
 await expect(api("/teams")).rejects.toBeInstanceOf(ApiError);
 expect(dispatchEvent).toHaveBeenCalledOnce();
 expect(dispatchEvent.mock.calls[0][0].type).toBe("viewer-access-required");
});
