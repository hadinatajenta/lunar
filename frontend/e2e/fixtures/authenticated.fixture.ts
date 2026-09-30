import { test as base } from "@playwright/test"

export type AuthenticatedFixtures = {
  authenticatedPage: typeof base extends (args: infer A) => unknown ? A : never
}

export const test = base.extend<Record<string, never>>({})
export { expect } from "@playwright/test"
