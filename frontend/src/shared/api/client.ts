import createClient from "openapi-fetch";

import type { components, paths } from "./openapi.gen";

export const apiClient = createClient<paths>({ baseUrl: "" });

export class ServerError extends Error {}

export function apiError(
  error: components["schemas"]["ErrorModel"] | undefined,
  fallback: Error,
): Error {
  return error?.detail == null ? fallback : new ServerError(error.detail);
}
