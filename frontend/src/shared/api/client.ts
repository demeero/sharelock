import createClient from "openapi-fetch";

import type { components, paths } from "./openapi.gen";

export const apiClient = createClient<paths>({ baseUrl: "" });

export function apiError(
  error: components["schemas"]["ErrorModel"] | undefined,
  fallback: string,
): Error {
  return new Error(error?.detail ?? fallback);
}
