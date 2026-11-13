export interface Action {
  kind: string;
  selector?: string;
  value?: string;
  at: number;
  redacted?: boolean;
}
export interface Fixture {
  id: string;
  method: string;
  url: string;
  requestBody?: string;
  status: number;
  body: string;
  contentType: string;
  delayMs: number;
  redacted?: boolean;
}
export interface Expectation {
  kind: string;
  selector?: string;
  value?: string;
}
export interface Variant {
  id: string;
  name: string;
  fixtureId: string;
  body: string;
  status: number;
  delayMs: number;
}
export interface BugCase {
  executionProblems?: string[];
  recordingState?: string;
  truncated?: boolean;
  version: number;
  id: string;
  title: string;
  createdAt: string;
  origin: string;
  width: number;
  height: number;
  events: unknown[];
  actions: Action[];
  fixtures: Fixture[];
  expectations: Expectation[];
  diagnostics: { kind: string; message: string; at: number }[];
  warnings: string[];
  variants: Variant[];
}
export interface Summary {
  id: string;
  title: string;
  origin: string;
  createdAt: string;
  actions: number;
  fixtures: number;
}
export interface Run {
  id: string;
  caseId: string;
  target: string;
  status: string;
  durationMs: number;
  steps: { label: string; status: string; detail?: string }[];
  problems: string[];
  screenshot?: string;
}
export interface Comparison {
  status: string;
  before: Run;
  after: Run;
}
export type Evidence = Run | Comparison;
export const isComparison = (value: Evidence): value is Comparison =>
  "before" in value;
