export type Language = "en" | "uk";

const ukPlurals = new Intl.PluralRules("uk");
function plural(
  language: Language,
  count: number,
  en: [string, string],
  uk: [string, string, string],
): string {
  const word =
    language === "en"
      ? en[count === 1 ? 0 : 1]
      : uk[ukPlurals.select(count) === "one" ? 0 : ukPlurals.select(count) === "few" ? 1 : 2];
  return `${count} ${word}`;
}

export function formatDuration(language: Language, seconds: number): string {
  if (seconds % 86400 === 0)
    return plural(language, seconds / 86400, ["day", "days"], ["день", "дні", "днів"]);
  if (seconds % 3600 === 0)
    return plural(language, seconds / 3600, ["hour", "hours"], ["година", "години", "годин"]);
  if (seconds % 60 === 0)
    return plural(language, seconds / 60, ["minute", "minutes"], ["хвилина", "хвилини", "хвилин"]);
  return plural(language, seconds, ["second", "seconds"], ["секунда", "секунди", "секунд"]);
}

export const en = {
  title: "Sharelock — encrypted temporary share",
  home: "Sharelock home",
  language: "Interface language",
  clientCrypto: "CLIENT-SIDE CRYPTO",
  footer: "AES-256-GCM · key stays in URL fragment · server stores ciphertext only",
  copied: "COPIED",
  copyFailed: "COPY FAILED",
  copy: "Copy",
  copyText: "Copy text",
  download: "Download",
  clipboardValue: "Clipboard access failed. Select the value and copy it manually.",
  clipboardText: "Clipboard access failed. Select the text and copy it manually.",
  clipboardContent: "Clipboard access failed. Select the content and copy it manually.",
  createNew: "Create a new share",
  remove: "Remove",
  payload: "PAYLOAD",
  fileText: "FILE + TEXT",
  file: "FILE",
  text: "TEXT",
  empty: "EMPTY",
  name: "Name",
  optional: "(optional)",
  accompanyingOptional: "Accompanying text (optional)",
  pasteText: "Paste text",
  textPlaceholder: "A password, config, certificate, or any other text",
  fileOptional: "FILE (OPTIONAL)",
  clearFile: "Clear file",
  replaceFile: "Replace file",
  chooseFile: "Choose file",
  noFile: "No file selected",
  accompanyingText: "Accompanying text",
  binary: "Binary payload. Use Download to save it locally.",
  unlimited: "unlimited opens",
  limitNotSet: "open limit not set",
  opens: (count: number) =>
    plural("en", count, ["open", "opens"], ["відкриття", "відкриття", "відкриттів"]),
  payloads: (count: number) =>
    plural("en", count, ["payload", "payloads"], ["елемент", "елементи", "елементів"]),
  itemCount: (count: number) =>
    plural("en", count, ["ITEM", "ITEMS"], ["ЕЛЕМЕНТ", "ЕЛЕМЕНТИ", "ЕЛЕМЕНТІВ"]),
  expiresIn: (duration: string) => `expires in ${duration}`,
  duration: (seconds: number) => formatDuration("en", seconds),
  settingsFailed: "Could not load share settings.",
  invalidSize: "The server returned an invalid encrypted share size limit.",
  invalidLifetime: "The server returned an invalid maximum share lifetime.",
  invalidOpens: "The server returned an invalid maximum number of opens.",
  addContent: "Add at least one non-empty text value or file.",
  chooseOpens: (max: number) => `Choose between 1 and ${max} opens.`,
  enterPassword: "Enter a password to protect this share.",
  passwordMismatch: "The password confirmation does not match.",
  tooLargeExpected: (size: string, max: string) =>
    `The encrypted share would be ${size}. The server limit is ${max}.`,
  tooLarge: (size: string, max: string) =>
    `The encrypted share is ${size}. The server limit is ${max}.`,
  createError: "Could not create the share.",
  shareReady: "SHARE READY",
  ready: "READY",
  createTitle: "Create encrypted share",
  stored: "The ciphertext is stored. The decryption key exists only in the complete share URL.",
  shareURL: "Share URL",
  revokeURL: "Revoke URL",
  passwordRequired: "PASSWORD REQUIRED",
  passwordDelivery:
    "The recipient must enter the password you chose. Send it through a separate trusted channel; it is not part of this URL.",
  revokeCapability: "REVOKE CAPABILITY",
  admin: "ADMIN",
  revokeWarning: "Anyone holding this URL can permanently revoke the share. Store it separately.",
  oneTime: "ONE-TIME OUTPUT",
  linksWarning:
    "Neither URL can be recovered later. Send the share URL only through a trusted channel.",
  createAnother: "Create another share",
  encrypted: "ENCRYPTED",
  createIntro:
    "Paste text, attach a file, or include both. Encryption happens before data leaves this browser.",
  privacy: "Privacy model",
  privacyBefore: "The share URL includes a decryption key after",
  privacyAfter:
    ". That fragment never reaches Sharelock; send the complete URL only through a trusted channel.",
  loadingSettings: "Loading server settings…",
  settingsUnavailable: "SETTINGS UNAVAILABLE",
  retry: "Retry",
  addPayload: "+ Add payload",
  deliveryPolicy: "DELIVERY POLICY",
  expiresAfter: "Expires after",
  limitOpens: "Limit number of opens",
  opensAllowed: "Opens allowed",
  lastOpenDeletion: "The share is deleted from the server on its last allowed open.",
  protectPassword: "Protect with a separate password",
  password: "Password",
  confirmPassword: "Confirm password",
  hidePasswords: "Hide passwords",
  showPasswords: "Show passwords",
  passwordPrivacy:
    "The password is used only in this browser and is never included in the share URL or upload.",
  createFailed: "CREATE FAILED",
  resetForm: "Reset form",
  encrypting: "Encrypting…",
  encryptCreate: "Encrypt & create",
  unavailable: "UNAVAILABLE",
  invalidKey: "INVALID KEY",
  passwordInvalid: "PASSWORD INVALID",
  missingKey: "This URL does not contain a decryption key.",
  shareUnavailable: "This share is unavailable, expired, burned, or revoked.",
  decryptError: "Could not decrypt this share.",
  incorrectPassword: "The password is incorrect or the password verifier is damaged.",
  damagedPayload: "The decryption key is invalid or the encrypted payload is damaged.",
  alreadyUnavailable: "This share is already unavailable.",
  revokeError: "Could not revoke this share.",
  revoked: "REVOKED",
  revokedMessage: "The encrypted blob was permanently removed. This share can no longer be opened.",
  revokeTitle: "Revoke encrypted share",
  revokeIntro:
    "This administrative capability permanently removes the encrypted blob from Sharelock.",
  danger: "DANGER",
  revokeConfirmWarning: "Revocation cannot be undone. Confirm the exact share before continuing.",
  revokeShare: "Revoke share",
  confirmRevoke: "Confirm permanent revoke",
  irreversible: "IRREVERSIBLE",
  revokeConsequences:
    "The ciphertext will be deleted immediately. Recipients will lose access even if they still have the share URL.",
  revokeFailed: "REVOKE FAILED",
  cancel: "Cancel",
  revoking: "Revoking…",
  revokePermanently: "Revoke permanently",
  decrypted: "DECRYPTED",
  decryptedTitle: "Decrypted payloads",
  plaintextWarning: "Plaintext exists only in this browser tab. Save what you need, then close it.",
  burned: "BURNED",
  burnedMessage: "That was the last allowed open. This share was deleted from the server.",
  opensLeft: "OPENS LEFT",
  remainingOpens: (count: number) =>
    `This share can be opened ${count} more ${count === 1 ? "time" : "times"} before it is deleted.`,
  sealed: "SEALED",
  openTitle: "Open encrypted share",
  openIntro:
    "The server returns ciphertext. The key in this URL decrypts it only inside this browser.",
  passwordIntro:
    "This share also requires a password. It is verified locally before an open is counted.",
  hidePassword: "Hide password",
  showPassword: "Show password",
  verifying: "Verifying…",
  verifyOpen: "Verify & open",
  preparing: "Preparing…",
  openDecrypt: "Open & decrypt",
};

export type Messages = typeof en;

export const uk: Messages = {
  title: "Sharelock — тимчасовий обмін зашифрованими секретами",
  home: "Головна Sharelock",
  language: "Мова інтерфейсу",
  clientCrypto: "ШИФРУВАННЯ В БРАУЗЕРІ",
  footer: "AES-256-GCM · ключ залишається у фрагменті URL · сервер зберігає лише зашифровані дані",
  copied: "СКОПІЙОВАНО",
  copyFailed: "НЕ ВДАЛОСЯ СКОПІЮВАТИ",
  copy: "Копіювати",
  copyText: "Копіювати текст",
  download: "Завантажити",
  clipboardValue: "Немає доступу до буфера обміну. Виділіть значення та скопіюйте його вручну.",
  clipboardText: "Немає доступу до буфера обміну. Виділіть текст і скопіюйте його вручну.",
  clipboardContent: "Немає доступу до буфера обміну. Виділіть вміст і скопіюйте його вручну.",
  createNew: "Створити новий секрет",
  remove: "Видалити",
  payload: "ЕЛЕМЕНТ",
  fileText: "ФАЙЛ + ТЕКСТ",
  file: "ФАЙЛ",
  text: "ТЕКСТ",
  empty: "ПОРОЖНЬО",
  name: "Назва",
  optional: "(необов’язково)",
  accompanyingOptional: "Супровідний текст (необов’язково)",
  pasteText: "Вставте текст",
  textPlaceholder: "Пароль, конфігурація, сертифікат або будь-який інший текст",
  fileOptional: "ФАЙЛ (НЕОБОВ’ЯЗКОВО)",
  clearFile: "Прибрати файл",
  replaceFile: "Замінити файл",
  chooseFile: "Вибрати файл",
  noFile: "Файл не вибрано",
  accompanyingText: "Супровідний текст",
  binary: "Двійкові дані. Натисніть «Завантажити», щоб зберегти їх локально.",
  unlimited: "необмежена кількість відкриттів",
  limitNotSet: "ліміт відкриттів не задано",
  opens: (count: number) =>
    plural("uk", count, ["open", "opens"], ["відкриття", "відкриття", "відкриттів"]),
  payloads: (count: number) =>
    plural("uk", count, ["payload", "payloads"], ["елемент", "елементи", "елементів"]),
  itemCount: (count: number) =>
    plural("uk", count, ["ITEM", "ITEMS"], ["ЕЛЕМЕНТ", "ЕЛЕМЕНТИ", "ЕЛЕМЕНТІВ"]),
  expiresIn: (duration: string) => `термін дії: ${duration}`,
  duration: (seconds: number) => formatDuration("uk", seconds),
  settingsFailed: "Не вдалося завантажити налаштування сервера.",
  invalidSize: "Сервер повернув некоректний ліміт розміру зашифрованого секрету.",
  invalidLifetime: "Сервер повернув некоректний максимальний термін дії секрету.",
  invalidOpens: "Сервер повернув некоректну максимальну кількість відкриттів.",
  addContent: "Додайте хоча б один непорожній текст або файл.",
  chooseOpens: (max: number) => `Вкажіть кількість відкриттів від 1 до ${max}.`,
  enterPassword: "Введіть пароль для захисту секрету.",
  passwordMismatch: "Паролі не збігаються.",
  tooLargeExpected: (size: string, max: string) =>
    `Розмір зашифрованого секрету становитиме ${size}. Ліміт сервера — ${max}.`,
  tooLarge: (size: string, max: string) =>
    `Розмір зашифрованого секрету — ${size}. Ліміт сервера — ${max}.`,
  createError: "Не вдалося створити секрет.",
  shareReady: "СЕКРЕТ ГОТОВИЙ",
  ready: "ГОТОВО",
  createTitle: "Створити зашифрований секрет",
  stored:
    "Зашифровані дані збережено. Ключ розшифрування міститься лише в повному посиланні на секрет.",
  shareURL: "Посилання на секрет",
  revokeURL: "Посилання для відкликання",
  passwordRequired: "ПОТРІБЕН ПАРОЛЬ",
  passwordDelivery:
    "Одержувач має ввести обраний вами пароль. Передайте його окремим довіреним каналом: він не є частиною цього посилання.",
  revokeCapability: "ВІДКЛИКАННЯ СЕКРЕТУ",
  admin: "АДМІН",
  revokeWarning:
    "Кожен, хто має це посилання, може назавжди відкликати секрет. Зберігайте його окремо.",
  oneTime: "ПОКАЗУЄТЬСЯ ОДИН РАЗ",
  linksWarning:
    "Відновити ці посилання пізніше неможливо. Передавайте посилання на секрет лише довіреним каналом.",
  createAnother: "Створити ще один секрет",
  encrypted: "ЗАШИФРОВАНО",
  createIntro:
    "Вставте текст, додайте файл або поєднайте їх. Дані шифруються до того, як залишають цей браузер.",
  privacy: "Приватність",
  privacyBefore: "Посилання на секрет містить ключ розшифрування після",
  privacyAfter:
    ". Цей фрагмент ніколи не надсилається до Sharelock; передавайте повне посилання лише довіреним каналом.",
  loadingSettings: "Завантаження налаштувань сервера…",
  settingsUnavailable: "НАЛАШТУВАННЯ НЕДОСТУПНІ",
  retry: "Спробувати ще раз",
  addPayload: "+ Додати елемент",
  deliveryPolicy: "УМОВИ ДОСТУПУ",
  expiresAfter: "Термін дії",
  limitOpens: "Обмежити кількість відкриттів",
  opensAllowed: "Дозволено відкриттів",
  lastOpenDeletion: "Секрет видаляється із сервера після останнього дозволеного відкриття.",
  protectPassword: "Захистити окремим паролем",
  password: "Пароль",
  confirmPassword: "Підтвердьте пароль",
  hidePasswords: "Приховати паролі",
  showPasswords: "Показати паролі",
  passwordPrivacy:
    "Пароль використовується лише в цьому браузері й ніколи не додається до посилання або даних, що надсилаються.",
  createFailed: "НЕ ВДАЛОСЯ СТВОРИТИ",
  resetForm: "Очистити форму",
  encrypting: "Шифрування…",
  encryptCreate: "Зашифрувати й створити",
  unavailable: "НЕДОСТУПНО",
  invalidKey: "НЕКОРЕКТНИЙ КЛЮЧ",
  passwordInvalid: "НЕКОРЕКТНИЙ ПАРОЛЬ",
  missingKey: "Це посилання не містить ключа розшифрування.",
  shareUnavailable:
    "Секрет недоступний: термін дії сплив, відкриття вичерпано або доступ відкликано.",
  decryptError: "Не вдалося розшифрувати секрет.",
  incorrectPassword: "Пароль неправильний або дані для його перевірки пошкоджено.",
  damagedPayload: "Ключ розшифрування некоректний або зашифровані дані пошкоджено.",
  alreadyUnavailable: "Цей секрет уже недоступний.",
  revokeError: "Не вдалося відкликати секрет.",
  revoked: "ВІДКЛИКАНО",
  revokedMessage: "Зашифровані дані видалено назавжди. Цей секрет більше неможливо відкрити.",
  revokeTitle: "Відкликати зашифрований секрет",
  revokeIntro: "Це адміністративне посилання назавжди видаляє зашифровані дані із Sharelock.",
  danger: "НЕБЕЗПЕКА",
  revokeConfirmWarning:
    "Відкликання неможливо скасувати. Перед продовженням переконайтеся, що це потрібний секрет.",
  revokeShare: "Відкликати секрет",
  confirmRevoke: "Підтвердьте остаточне відкликання",
  irreversible: "БЕЗПОВОРОТНО",
  revokeConsequences:
    "Зашифровані дані буде негайно видалено. Одержувачі втратять доступ, навіть якщо збережуть посилання на секрет.",
  revokeFailed: "НЕ ВДАЛОСЯ ВІДКЛИКАТИ",
  cancel: "Скасувати",
  revoking: "Відкликання…",
  revokePermanently: "Відкликати назавжди",
  decrypted: "РОЗШИФРОВАНО",
  decryptedTitle: "Розшифровані елементи",
  plaintextWarning:
    "Розшифровані дані є лише в цій вкладці браузера. Збережіть потрібне й закрийте її.",
  burned: "ВІДКРИТТЯ ВИЧЕРПАНО",
  burnedMessage: "Це було останнє дозволене відкриття. Секрет видалено із сервера.",
  opensLeft: "ЗАЛИШИЛОСЯ ВІДКРИТТІВ",
  remainingOpens: (count: number) =>
    `До видалення цей секрет можна відкрити ще ${plural("uk", count, ["time", "times"], ["раз", "рази", "разів"])}.`,
  sealed: "ЗАХИЩЕНО",
  openTitle: "Відкрити зашифрований секрет",
  openIntro:
    "Сервер повертає зашифровані дані. Ключ у цьому посиланні розшифровує їх лише всередині браузера.",
  passwordIntro:
    "Для цього секрету також потрібен пароль. Він перевіряється локально до зарахування відкриття.",
  hidePassword: "Приховати пароль",
  showPassword: "Показати пароль",
  verifying: "Перевірка…",
  verifyOpen: "Перевірити й відкрити",
  preparing: "Підготовка…",
  openDecrypt: "Відкрити й розшифрувати",
};

export const translations: Record<Language, Messages> = { en, uk };
