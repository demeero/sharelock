import { mount } from "svelte";

import App from "./App.svelte";
import "./styles/app.css";
import { locale } from "./shared/i18n/locale";
import { translations } from "./shared/i18n/messages";

locale.subscribe(({ language }) => {
  document.documentElement.lang = language;
  document.title = translations[language].title;
});

const target = document.querySelector<HTMLDivElement>("#app");
if (!target) {
  throw new Error("Sharelock application target is missing.");
}

mount(App, { target });
