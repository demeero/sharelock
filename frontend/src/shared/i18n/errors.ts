import { ServerError } from "../api/client";
import { en, type Messages } from "./messages";

// Keep a message recipe so an already-visible error follows language changes.
export class LocalizedError extends Error {
  constructor(readonly render: (messages: Messages) => string) {
    super(render(en));
  }
}

export function errorMessage(error: unknown, messages: Messages, fallback: string): string {
  if (error instanceof LocalizedError) return error.render(messages);
  if (error instanceof ServerError) return error.message;
  return fallback;
}
