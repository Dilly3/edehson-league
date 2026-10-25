import { describe, expect, it } from "vitest";
import { resolvePage, isAdminPath, pageURL } from "./navigation";
describe("separate admin routes", () => {
  it("opens GreatMindsSport at the roster and preserves league links", () => {
    expect(resolvePage("/", "")).toBe("membership");
    expect(resolvePage("/", "#league")).toBe("overview");
    expect(resolvePage("/", "#overview")).toBe("overview");
    expect(resolvePage("/", "#fixtures")).toBe("fixtures");
  });
  it("opens the dashboard at either admin path", () => {
    expect(resolvePage("/admin", "")).toBe("admin");
    expect(resolvePage("/admin/", "")).toBe("admin");
    expect(isAdminPath("/administrator")).toBe(false);
  });
  it("keeps management navigation within admin", () => {
    expect(pageURL("teams", true)).toBe("/admin#teams");
    expect(resolvePage("/admin", "#teams")).toBe("teams");
    expect(pageURL("admin", false)).toBe("/admin");
  });
  it("does not resolve the old public admin hash", () => {
    expect(resolvePage("/", "#admin")).toBe("membership");
    expect(resolvePage("/", "#teams")).toBe("teams");
    expect(pageURL("teams", false)).toBe("/#teams");
  });
});
