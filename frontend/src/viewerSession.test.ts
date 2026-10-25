import { afterEach, describe, expect, it, vi } from "vitest";
import { createViewerIdleTimer, VIEWER_IDLE_MS } from "./viewerSession";
afterEach(()=>vi.useRealTimers());
describe("viewer idle session",()=>{
 it("logs out at two hours without activity",()=>{
  vi.useFakeTimers();const onIdle=vi.fn(),onActivity=vi.fn();createViewerIdleTimer({onIdle,onActivity});
  vi.advanceTimersByTime(VIEWER_IDLE_MS-1000);expect(onIdle).not.toHaveBeenCalled();
  vi.advanceTimersByTime(1000);expect(onIdle).toHaveBeenCalledOnce();expect(onActivity).not.toHaveBeenCalled();
 });
 it("renews only on actual activity and throttles heartbeat requests",()=>{
  vi.useFakeTimers();vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));const onIdle=vi.fn(),onActivity=vi.fn();const timer=createViewerIdleTimer({onIdle,onActivity});
  vi.advanceTimersByTime(60*60*1000);timer.activity();expect(onActivity).toHaveBeenCalledOnce();
  timer.activity();timer.activity();expect(onActivity).toHaveBeenCalledOnce();
  vi.advanceTimersByTime(60000);expect(onActivity).toHaveBeenCalledTimes(2);
  vi.advanceTimersByTime(VIEWER_IDLE_MS-61000);expect(onIdle).not.toHaveBeenCalled();vi.advanceTimersByTime(1000);expect(onIdle).toHaveBeenCalledOnce();
 });
 it("uses the server's remaining time and cannot revive an expired session",()=>{
  vi.useFakeTimers();const onIdle=vi.fn(),onActivity=vi.fn();const timer=createViewerIdleTimer({remaining:1000,onIdle,onActivity});
  vi.advanceTimersByTime(1000);expect(timer.activity()).toBe(false);expect(onIdle).toHaveBeenCalledOnce();expect(onActivity).not.toHaveBeenCalled();
 });
 it("shares activity between tabs without duplicate server requests",()=>{
  vi.useFakeTimers();const onIdle=vi.fn(),onActivity=vi.fn();const timer=createViewerIdleTimer({remaining:1000,onIdle,onActivity});
  timer.activity(false);vi.advanceTimersByTime(2000);expect(onIdle).not.toHaveBeenCalled();expect(onActivity).not.toHaveBeenCalled();timer.stop();
 });
});
