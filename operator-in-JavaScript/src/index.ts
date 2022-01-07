/* eslint-disable @typescript-eslint/no-explicit-any */
import * as k8s from "@kubernetes/client-node";
import * as fs from "fs";
import axios from "axios";
import { Completion } from "./types";
import { OPENAI_API_KEY } from "./constants";

const COMPLETION_GROUP = "copilot.poc.com";
const COMPLETION_VERSION = "v1";
const COMPLETION_PLURAL = "completions";

const kc = new k8s.KubeConfig();
kc.loadFromDefault();

// const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
const k8sApiMC = kc.makeApiClient(k8s.CustomObjectsApi);

const watch = new k8s.Watch(kc);

async function onEvent(phase: string, apiObj: any) {
  console.log(`Received event in phase ${phase}.`);
  if (phase == "ADDED" || phase == "MODIFIED") {
    scheduleReconcile(apiObj);
  } else if (phase == "DELETED") {
    console.log(`Deleted ${apiObj.metadata.name}`);
  } else {
    console.log(`Unknown event type: ${phase}`);
  }
}

function onDone(err: any) {
  console.log(`Connection closed`);
  if (typeof err !== "undefined") {
    console.log("got err: ", err);
    process.exit();
  }
  watchResource();
}

async function watchResource(): Promise<any> {
  console.log("Watching API");
  return watch.watch(`/apis/${COMPLETION_GROUP}/${COMPLETION_VERSION}/${COMPLETION_PLURAL}`, {}, onEvent, onDone);
}

let reconcileScheduled = false;

function scheduleReconcile(obj: Completion) {
  if (!reconcileScheduled) {
    setTimeout(reconcileNow, 1000, obj);
    reconcileScheduled = true;
  }
}

function needsReconcile(obj: Completion) {
  return obj.status!.observedGeneration !== obj.metadata.generation;
}

async function reconcileNow(obj: Completion) {
  reconcileScheduled = false;
  if (!obj.status) {
    console.log("No status for object, setting a new one");
    obj.status = {
      completion: "",
      observedGeneration: typeof obj.metadata.generation !== "undefined" ? obj.metadata.generation : 1,
    };
  }
  if (!needsReconcile(obj)) {
    return;
  }
  if (!obj.spec) {
    console.log("error: no spec");
    return;
  }

  // const userData = obj.spec?.userPrompt;
  const { userPrompt, maxTokens } = obj.spec!;
  if (typeof userPrompt !== "string") {
    console.error("user data is not a string");
    return;
  }

  try {
    const openAIUrl = "https://api.openai.com/v1/engines/davinci-codex/completions";
    // request openai API to complete data

    const headers = {
      "Authorization": `Bearer ${OPENAI_API_KEY}`,
      "Content-Type": "application/json",
    };
    const body = {
      prompt: "# Below is a series of YAML files used to create resources in a Kubernetes cluster\n" + userPrompt,
      max_tokens: typeof maxTokens !== "undefined" ? maxTokens : 64,
      stop: ["#\n#\n", "\n\n---\n\n", "\n\n"],
      temperature: 0.12,
      top_p: 1,
      frequency_penalty: 0,
      presence_penalty: 0,
    };

    // request the openai api using axios
    await axios.post(openAIUrl, body, { headers }).then(async (response) => {
      // update the object with the competion result
      if (response.status == 200 && response.data.choices) {
        const completion = response.data.choices[0]!.text;
        obj.status!.completion = completion;
        // we increment the observedGeneration field here since this updates the object
        obj.status!.observedGeneration = obj.metadata.generation! + 1;
        await k8sApiMC.replaceClusterCustomObject(
          COMPLETION_GROUP,
          COMPLETION_VERSION,
          COMPLETION_PLURAL,
          obj.metadata.name!,
          obj,
        );
      }
    });
  } catch (error) {
    console.log("error while reconciling object: ", error);
    // save the error to a json file
    const errorFile = `${obj.metadata.name}.json`;
    fs.writeFileSync(errorFile, JSON.stringify(error));
    console.log("wrote to file");
  }
}

async function main() {
  await watchResource();
}

main().catch((err) => {
  console.log(err);
});
