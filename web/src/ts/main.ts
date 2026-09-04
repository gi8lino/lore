// Browser entry point that initializes Lore UI features.

import { initAdmin } from "./features/admin/index.ts";
import { initEditor } from "./features/editor/index.ts";
import { initExports } from "./features/exports.ts";
import { initLayout } from "./features/layout.ts";
import { initMarkdown } from "./features/markdown.ts";
import { initMedia } from "./features/media.ts";
import { initPage } from "./features/page.ts";
import { initSearch } from "./features/search.ts";
import { initTokens } from "./features/tokens.ts";
import { initConfirmForms } from "./core/dialogs.ts";
import { initCommandPalette } from "./core/command-palette.ts";
import { initTheme } from "./core/theme.ts";
import { initKeyboardWorkflow } from "./core/keyboard.ts";
import { initPWA } from "./core/pwa.ts";
import { initDashboard } from "./features/dashboard.ts";
import { initGraph } from "./features/graph.ts";
import { initNotifications } from "./features/notifications.ts";
import { initMentionAutocomplete } from "./features/mentions.ts";

initTheme();
initPWA();
initLayout();
initKeyboardWorkflow();
initSearch();
initEditor();
initMedia();
initTokens();
initExports();
initPage();
initDashboard();
initGraph();
initNotifications();
initMentionAutocomplete();
initConfirmForms();
initCommandPalette();
await initMarkdown();
initAdmin();
