# TallerFlow Frontend Roles and Client Portal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver usable, responsive frontend flows for OWNER, ADMIN, OPERATOR and CLIENT without bypassing backend authorization.

**Architecture:** Keep internal and client sessions distinct. Reuse the Vue Router, Pinia session store and shared UI components; add a small role-aware app shell and a dedicated client-portal module. The internal UI sends CSRF for every mutation and renders only actions supported by the current backend contract.

**Tech Stack:** Vue 3, TypeScript, Pinia, Vue Router, Vitest, Testing Library, existing UI components and CSS.

**Spec:** `docs/superpowers/specs/2026-10-02-tallerflow-frontend-roles-client-portal-design.md`

## Global Constraints

- Keep the brand `TallerFlow`, Manrope, `#0D9488`, `#0F172A`, `#0284C7` and `#64748B` from the approved Stitch guide.
- Never trust a frontend role guard as authorization; backend responses remain authoritative.
- Internal `POST`, `PATCH` and `DELETE` requests send the current session CSRF token.
- Client-portal code must not use or overwrite the internal Pinia session.
- Do not enable production actions for pause, cancellation, correction or assignment before backend endpoints and audit rules exist.
- All new views have loading, empty, error and keyboard-accessible states, and work at 360 px.

## Review Focus

- A direct route to an OWNER-only screen as OPERATOR returns the user to the appropriate internal screen without rendering restricted controls.
- A lost/expired CSRF token makes a visible recoverable error and never submits a mutation without the header.
- An empty client list prevents order creation with a clear path back to create a client.
- A client portal response cannot render an internal client, workshop member or private note.
- A 409 conflict while advancing a stage refreshes the order rather than showing stale progress.

---

### Task 1: Establish the role-aware internal application shell

**Files:**
- Create: `frontend/src/modules/session/AppShell.vue`
- Create: `frontend/src/modules/session/role-policy.ts`
- Create: `frontend/src/modules/session/role-policy.test.ts`
- Modify: `frontend/src/modules/session/SessionHomeView.vue`
- Modify: `frontend/src/app/router.ts`

**Interfaces:**
- Produces `canManageOperations(role: string | undefined): boolean` and `canAdvanceProduction(role: string | undefined): boolean`.
- Produces `AppShell` slots `header`, `navigation` and `default` for internal screens.

- [ ] **Step 1: Write failing role-policy tests**

```ts
expect(canManageOperations('OWNER')).toBe(true)
expect(canManageOperations('ADMIN')).toBe(true)
expect(canManageOperations('OPERATOR')).toBe(false)
expect(canAdvanceProduction('OPERATOR')).toBe(true)
```

- [ ] **Step 2: Run the role-policy test and verify it fails because the module is absent**

Run: `npm test -- src/modules/session/role-policy.test.ts`

- [ ] **Step 3: Implement the role policy and accessible `AppShell`**

Keep links limited to Dashboard, Clients, Orders, New Order and Team according to the policy. Make `SessionHomeView` use the shell rather than reimplementing navigation.

- [ ] **Step 4: Run the focused tests and production build**

Run: `npm test -- src/modules/session/role-policy.test.ts` and `npm run build`

- [ ] **Step 5: Commit the role shell**

```bash
git add frontend/src/modules/session frontend/src/app/router.ts
git commit -m "feat(frontend): add role-aware internal shell"
```

### Task 2: Finish OWNER and ADMIN client and order flows

**Files:**
- Modify: `frontend/src/modules/clients/ClientsView.vue`
- Modify: `frontend/src/modules/clients/api.ts`
- Modify: `frontend/src/modules/orders/OrderCreateView.vue`
- Modify: `frontend/src/modules/orders/OrdersListView.vue`
- Modify: `frontend/src/modules/orders/api.ts`
- Create: `frontend/src/modules/clients/ClientsView.test.ts`
- Create: `frontend/src/modules/orders/OrderCreateView.test.ts`
- Create: `frontend/src/modules/orders/OrdersListView.test.ts`

**Interfaces:**
- Consumes `canManageOperations` and `session.csrfToken` from Task 1.
- Produces client and order mutations that always pass the CSRF token to `clientsApi` and `ordersApi`.

- [ ] **Step 1: Write failing view tests**

Test that an OWNER creates a client with the CSRF token, an OPERATOR cannot see the create-client or new-order controls, and an empty client list presents a create-client action instead of an invalid order form.

- [ ] **Step 2: Run the focused tests and verify the missing permission/CSRF behavior fails**

Run: `npm test -- src/modules/clients/ClientsView.test.ts src/modules/orders/OrderCreateView.test.ts src/modules/orders/OrdersListView.test.ts`

- [ ] **Step 3: Implement forms, filters and state screens**

Use the approved Stitch hierarchy: search and filters first, table/cards second, one clear primary action. Pass `session.csrfToken ?? undefined` to all internal mutation API calls. Preserve current list filters when returning from detail.

- [ ] **Step 4: Run focused tests and the frontend suite**

Run: `npm test -- src/modules/clients/ClientsView.test.ts src/modules/orders/OrderCreateView.test.ts src/modules/orders/OrdersListView.test.ts` and `npm test`

- [ ] **Step 5: Commit the owner/admin operations screens**

```bash
git add frontend/src/modules/clients frontend/src/modules/orders
git commit -m "feat(frontend): complete client and order operations"
```

### Task 3: Deliver the operator production queue and safe detail UI

**Files:**
- Create: `frontend/src/modules/orders/OperatorQueueView.vue`
- Modify: `frontend/src/modules/orders/OrderDetailView.vue`
- Modify: `frontend/src/modules/orders/api.ts`
- Modify: `frontend/src/app/router.ts`
- Create: `frontend/src/modules/orders/OperatorQueueView.test.ts`
- Create: `frontend/src/modules/orders/OrderDetailView.test.ts`

**Interfaces:**
- Consumes `canAdvanceProduction` and `session.csrfToken` from Task 1.
- Consumes `ordersApi.list`, `ordersApi.getByID` and `ordersApi.updateStage`.
- Produces `/app/production` as the OPERATOR landing route and an order detail that exposes only the current-stage action.

- [ ] **Step 1: Write failing operator tests**

Test that the queue renders active work and that the detail has one action for an `IN_PROGRESS` stage, passes CSRF, and hides administration controls from an OPERATOR.

- [ ] **Step 2: Run focused tests and verify the production queue is absent**

Run: `npm test -- src/modules/orders/OperatorQueueView.test.ts src/modules/orders/OrderDetailView.test.ts`

- [ ] **Step 3: Implement queue, role redirect and guarded stage action**

Do not provide UI controls to start arbitrary pending stages or reopen completed ones. If the API returns `409`, reload the detail and explain that another update occurred.

- [ ] **Step 4: Run focused tests and production build**

Run: `npm test -- src/modules/orders/OperatorQueueView.test.ts src/modules/orders/OrderDetailView.test.ts` and `npm run build`

- [ ] **Step 5: Commit the operator workflow**

```bash
git add frontend/src/modules/orders frontend/src/app/router.ts
git commit -m "feat(frontend): add operator production workflow"
```

### Task 4: Add the isolated client-portal frontend contract and screens

**Files:**
- Create: `frontend/src/modules/client-portal/api.ts`
- Create: `frontend/src/modules/client-portal/types.ts`
- Create: `frontend/src/modules/client-portal/ClientPortalAccessView.vue`
- Create: `frontend/src/modules/client-portal/ClientPortalOrdersView.vue`
- Create: `frontend/src/modules/client-portal/ClientPortalOrderDetailView.vue`
- Modify: `frontend/src/app/router.ts`
- Create: `frontend/src/modules/client-portal/ClientPortalAccessView.test.ts`
- Create: `frontend/src/modules/client-portal/ClientPortalOrdersView.test.ts`
- Create: `frontend/src/modules/client-portal/ClientPortalOrderDetailView.test.ts`

**Interfaces:**
- Produces `clientPortalApi.requestAccess(email)`, `listOrders()` and `getOrder(id)`.
- Produces `/portal/access`, `/portal/orders` and `/portal/orders/:id`; these never import `useSessionStore`.

- [ ] **Step 1: Write failing portal tests**

Test the anti-enumeration confirmation after a link request, empty orders, a private-detail timeline, and that no internal member or note fields appear in the portal types or DOM.

- [ ] **Step 2: Run focused tests and verify the portal module is absent**

Run: `npm test -- src/modules/client-portal/ClientPortalAccessView.test.ts src/modules/client-portal/ClientPortalOrdersView.test.ts src/modules/client-portal/ClientPortalOrderDetailView.test.ts`

- [ ] **Step 3: Implement typed read-only portal views**

Show a useful unavailable/error state until the backend endpoints are delivered. Do not mock privileged data or call internal `/api/v1/orders` endpoints from portal screens.

- [ ] **Step 4: Run focused tests and the frontend suite**

Run: `npm test -- src/modules/client-portal` and `npm test`

- [ ] **Step 5: Commit the client portal frontend**

```bash
git add frontend/src/modules/client-portal frontend/src/app/router.ts
git commit -m "feat(frontend): add read-only client portal"
```

### Task 5: Harden routes, responsive layout and final verification

**Files:**
- Modify: `frontend/src/app/router.test.ts`
- Modify: `frontend/src/styles/main.css`
- Create: `frontend/src/app/role-navigation.test.ts`
- Create: `frontend/src/modules/session/AppShell.test.ts`
- Create: `frontend/e2e/roles-and-client-portal.spec.ts`

**Interfaces:**
- Consumes all routes and role policy from Tasks 1–4.
- Produces regression coverage for role redirect, internal CSRF call sites, portal isolation and 360 px viewport behavior.

- [ ] **Step 1: Write failing navigation and responsive tests**

Cover direct protected navigation, owner/admin menus, operator menu, portal route isolation and an error state at 360 px.

- [ ] **Step 2: Run tests and verify the current router does not meet the new contracts**

Run: `npm test -- src/app/router.test.ts src/app/role-navigation.test.ts`

- [ ] **Step 3: Implement final router guards and responsive style adjustments**

Keep `/portal/*` public only where necessary for client session restoration and redirect authenticated internal users away from portal-only login screens when appropriate.

- [ ] **Step 4: Run all frontend verification commands**

Run: `npm test`, `npm run build`, and `npm run test:e2e`

- [ ] **Step 5: Commit verification and accessibility coverage**

```bash
git add frontend/src frontend/e2e
git commit -m "test(frontend): cover roles and client portal navigation"
```

## Implementation Boundary

This plan intentionally delivers frontend screens and contracts first. The separate backend phase must add the client portal session, magic-link delivery, tenant-scoped portal reads, CSRF enforcement, ETag conflict handling and atomic stage transitions before a production client flow is enabled.
