import { mount } from "svelte";

import App from "./App.svelte";
import "./styles/app.css";

const target = document.querySelector<HTMLDivElement>("#app");
if (!target) {
  throw new Error("Sharelock application target is missing.");
}

mount(App, { target });
