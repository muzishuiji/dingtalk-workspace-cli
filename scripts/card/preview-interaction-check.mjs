#!/usr/bin/env node

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const previewPath = process.argv[2];
if (!previewPath) {
  throw new Error("usage: preview-interaction-check.mjs <preview.html>");
}

const html = readFileSync(previewPath, "utf8");
const scripts = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)];
assert.ok(scripts.length > 0, "preview must contain an interaction script");

const textField = {
  type: "textarea",
  value: "本地交互验收通过",
  dataset: { bindingPath: "/form/comment" },
};
const firstChoice = {
  type: "radio",
  checked: false,
  value: "option_1",
  dataset: { bindingPath: "/form/choice" },
};
const secondChoice = {
  type: "radio",
  checked: true,
  value: "option_2",
  dataset: { bindingPath: "/form/choice" },
};
const primaryButton = {
  dataset: { eventName: "primary", surfaceId: "dws-local-approval" },
  addEventListener(type, listener) {
    assert.equal(type, "click");
    this.click = listener;
  },
};
const result = { hidden: true };
const output = { textContent: "" };
let dispatched;

global.document = {
  querySelectorAll(selector) {
    if (selector === "[data-binding-path]") {
      return [textField, firstChoice, secondChoice];
    }
    if (selector === "button[data-event-name]") {
      return [primaryButton];
    }
    throw new Error(`unexpected selector: ${selector}`);
  },
  getElementById(id) {
    if (id === "interaction-json") return output;
    if (id === "interaction-result") return result;
    throw new Error(`unexpected element id: ${id}`);
  },
  dispatchEvent(event) {
    dispatched = event;
  },
};
global.CustomEvent = class CustomEvent {
  constructor(type, init) {
    this.type = type;
    this.detail = init.detail;
  }
};

new Function(scripts.at(-1)[1])();
assert.equal(typeof primaryButton.click, "function", "button click listener must be registered");
primaryButton.click();

const payload = JSON.parse(output.textContent);
assert.deepEqual(payload, {
  event: {
    name: "primary",
    context: {
      surfaceId: "dws-local-approval",
      form: {
        comment: "本地交互验收通过",
        choice: "option_2",
      },
    },
  },
  localPreview: true,
});
assert.equal(result.hidden, false, "interaction result must become visible");
assert.equal(dispatched.type, "dws-a2ui-preview-action");
assert.deepEqual(dispatched.detail, payload);

process.stdout.write(`${JSON.stringify({ status: "PASS", payload })}\n`);
