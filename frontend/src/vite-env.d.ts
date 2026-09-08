/// <reference types="vite/client" />

declare module "svelte/elements" {
  interface HTMLAttributes<T> {
    "align-"?: string;
    "bg-"?: string;
    "box-"?: string;
    "cap-"?: string;
    "container-"?: string;
    "fg-"?: string;
    "gap-"?: string;
    "is-"?: string;
    "pad-"?: string;
    "position-"?: string;
    "self-"?: string;
    "shear-"?: string;
    "size-"?: string;
    "speed-"?: string;
    "variant-"?: string;
    "wrap-"?: boolean | string;
  }
}

export {};
