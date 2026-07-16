import { scanCurrentItem } from "./scpApi.js";

Office.onReady(() => {
  (globalThis as { scanCurrentItem?: typeof scanCurrentItem }).scanCurrentItem =
    scanCurrentItem;
});
