import { requestPost } from "@/utils/request";

export function queryAuditLogs(payload: {
  organizationId?: string;
  page?: number;
  pageSize?: number;
}) {
  return requestPost<{
    items: any[];
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  }>("/api/manage/v1/audit_log/query", payload);
}
