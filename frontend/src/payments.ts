import { api } from "./api";
import { cardQuote, type RegisterData } from "./membership";

export interface PaymentAccount { bank_name: string; account_name: string; account_number: string; updated_at?: string }
export interface PendingReceipt { id: string; months: string[]; yellow_count: number; red_count: number; amount: number; created_at: string }
export interface PaymentRequest {
  id: string; member_id: string; member_name: string; months: string[]; yellow_count: number; red_count: number;
  dues_amount: number; yellow_amount: number; red_amount: number; amount: number; amount_sent: number;
  paid_on: string; note: string; status: "pending" | "confirmed" | "rejected"; receipt_name: string; receipt_type: string;
  receipt_size: number; review_note: string; reviewed_at: string | null; reviewed_by: string; receipt_deleted_at: string | null; created_at: string;
}

export const getPaymentAccount = () => api<PaymentAccount | null>("/membership/payment-account");
export const savePaymentAccount = (token: string, a: PaymentAccount) =>
  api<PaymentAccount>("/membership/payment-account", { method: "PUT", body: JSON.stringify(a) }, token);

export const getPendingReceipts = (memberID: string) =>
  api<PendingReceipt[]>(`/membership/payment-requests/pending?member_id=${encodeURIComponent(memberID)}`);

export function submitReceipt(input: { memberID: string; months: string[]; yellow: number; red: number; amountSent: string; paidOn: string; note: string; file: File }) {
  const form = new FormData();
  form.set("member_id", input.memberID);
  form.set("months", input.months.join(","));
  form.set("yellow_count", String(input.yellow));
  form.set("red_count", String(input.red));
  form.set("amount_sent", input.amountSent.replace(/,/g, ""));
  form.set("paid_on", input.paidOn);
  form.set("note", input.note);
  form.set("receipt", input.file);
  return api<PaymentRequest>("/membership/payment-requests", { method: "POST", body: form });
}

export const listReceipts = (token: string, status: "pending" | "all", memberID?: string) =>
  api<PaymentRequest[]>(`/membership/payment-requests?status=${status}${memberID ? `&member_id=${encodeURIComponent(memberID)}` : ""}`, {}, token);
/** Super admin: delete one reviewed receipt's stored file (the record and payments stay). */
export const deleteReceiptFile = (token: string, id: string) =>
  api<void>(`/membership/payment-requests/${id}/receipt`, { method: "DELETE" }, token);
/** Super admin: delete stored files for receipts reviewed more than `days` days ago. */
export const purgeReceiptFiles = (token: string, days: number) =>
  api<{ deleted: number; failed: number; bytes_freed: number }>("/membership/payment-requests/purge-files", { method: "POST", body: JSON.stringify({ older_than_days: days }) }, token);
export const confirmReceipt = (token: string, id: string) =>
  api<void>(`/membership/payment-requests/${id}/confirm`, { method: "POST", body: "{}" }, token);
export const rejectReceipt = (token: string, id: string, reason: string) =>
  api<void>(`/membership/payment-requests/${id}/reject`, { method: "POST", body: JSON.stringify({ reason }) }, token);

/** Downloads a receipt with the admin credential and returns a temporary object URL. */
export async function receiptURL(token: string, id: string): Promise<string> {
  const response = await fetch(`/api/membership/payment-requests/${id}/receipt`, { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" });
  if (!response.ok) throw new Error(response.status === 410 ? "This receipt file was deleted to save storage." : response.status === 502 ? "The receipt file could not be loaded from storage." : "Could not open the receipt.");
  return URL.createObjectURL(await response.blob());
}

/** Price of `count` more cards of one colour after cards already claimed by pending receipts. */
export function extraCardPrice(data: RegisterData, memberID: string, color: "yellow" | "red", pending: number, count: number): number | null {
  if (count === 0) return 0;
  const withNew = cardQuote(data, memberID, color, pending + count);
  const already = pending ? cardQuote(data, memberID, color, pending) : 0;
  return withNew === null || already === null ? null : withNew - already;
}

/** Opens a receipt in a new tab (admin). */
export async function openReceipt(token: string, id: string) {
  const tab = window.open("", "_blank");
  try {
    const url = await receiptURL(token, id);
    if (tab) tab.location.href = url; else window.location.assign(url);
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  } catch (e) { tab?.close(); throw e; }
}

/** CSV of receipts for record keeping (amounts in naira). */
export function receiptsCSV(rows: PaymentRequest[]): string {
  const naira = (k: number) => (k / 100).toFixed(2);
  const esc = (v: string | number) => { const s = String(v); return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s; };
  const head = ["Submitted", "Player", "Months", "Yellow cards", "Red cards", "Expected (NGN)", "Sent (NGN)", "Transfer date", "Status", "Reviewed by", "Reviewed at", "Review note", "Player note", "Receipt file"];
  const lines = rows.map(r => [r.created_at, r.member_name, r.months.join(" "), r.yellow_count, r.red_count, naira(r.amount), naira(r.amount_sent), r.paid_on, r.status,
    r.reviewed_by, r.reviewed_at ?? "", r.review_note, r.note, r.receipt_deleted_at ? "deleted" : "stored"].map(esc).join(","));
  return [head.join(","), ...lines].join("\n");
}

export const getReceiptSettings = (token: string) => api<{ retention_days: number }>("/membership/receipt-settings", {}, token);
/** Super admin: how many days reviewed receipt files are kept before the weekly cleanup deletes them. */
export const saveReceiptSettings = (token: string, retentionDays: number) =>
  api<{ retention_days: number }>("/membership/receipt-settings", { method: "PUT", body: JSON.stringify({ retention_days: retentionDays }) }, token);

/** "90 days (about 3 months)" style label. */
export function retentionLabel(days: number): string {
  if (days % 365 === 0) return `${days / 365} year${days === 365 ? "" : "s"}`;
  if (days >= 28 && days % 30 === 0) return `${days} days (about ${days / 30} month${days === 30 ? "" : "s"})`;
  return `${days} day${days === 1 ? "" : "s"}`;
}
