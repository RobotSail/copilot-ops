import * as k8s from "@kubernetes/client-node";

export type Completion = {
  apiVersion: string;
  kind: string;
  metadata: k8s.V1ObjectMeta;
  spec?: CompletionSpec;
  status?: CompletionStatus;
};

export type CompletionSpec = {
  userPrompt: string;
  maxTokens: number;
  // temperature: number;
};

export type CompletionStatus = {
  completion: string;
  observedGeneration: number;
};
