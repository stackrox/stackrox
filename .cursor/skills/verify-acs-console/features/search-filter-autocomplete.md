# Search and filter autocomplete

Users filter ACS lists in two first-class ways: the **global Search** page (masthead) and **compound search filters** on Workload CVE (and other) toolbars. Autocomplete options come from Central (`searchAutocomplete` GraphQL / search metadata), not from hardcoded coordinates.

## Sub-features

- `search-page-open` opens `/main/search` with `h1` **Search** and placeholder **Filter resources**.
- `search-page-empty` shows inline alert **Enter a new search filter** when no filter is complete.
- `cve-compound-entity` opens the entity selector (`compound search filter entity selector toggle`).
- `cve-compound-autocomplete` types into `[aria-label^="Filter results by"]` and applies via **Apply autocomplete input to search**.
- `cve-filter-chip` shows the value under `aria-label="applied search filters"`.

## How to get to it (user POV)

- Global Search: masthead **Search** (Search icon button in `MastheadToolbar.tsx`) → `/main/search`.
- Compound filter: open User Workloads or Platform CVE list; use the toolbar entity/attribute selectors, then the typeahead.

## Driving it with Cypress e2e

Preconditions:

- Doctor READY.
- Search route requires a wide set of read permissions (`routePaths.ts` `search` key). Admin has them. Reduced roles may hide the masthead Search button — then skip `search-page-*` and say why.

- **Open Search.** Authenticated click on masthead `button` **Search**, or visit `/main/search`. Result: `#search-page`, `h1:contains("Search")`, input placeholder `Filter resources`. Empty complete filter → alert **Enter a new search filter**.
- **Global filter (react-select).** Global Search still uses `SearchFilterInput` / react-select (`.react-select__input > input`, `.react-select__option`) as in `cypress/selectors/search.js`. Pick a category from options, type a value, complete the chip. Result: URL gains query string; results table or **No results match the search filter**.
- **Compound entity/attribute.** On `/main/vulnerabilities/user-workloads`, `selectEntity('CVE')` then `selectAttribute(...)` from `cypress/helpers/compoundFilters.ts` (`[aria-label="compound search filter entity selector toggle"]` / `attribute selector toggle`).
- **Autocomplete apply.** `cy.get('[aria-label^="Filter results by"]').type(<value>)` then `cy.get('[aria-label="Apply autocomplete input to search"]').click()` (`addAutocompleteFilter`). Menu list is `[aria-label="Filter results select menu"]`. Result: chip in `.pf-v6-c-toolbar .pf-v6-c-toolbar__group[aria-label="applied search filters"]`; table or empty state updates; URL search params change.
- **Proof.** Two frames: typeahead open with suggestions (or “No options”), then chip applied. Chrome (nav + toolbar) visible. Record the request to search autocomplete / list query.

## Gotchas

- Two widgets, two selector families. Do not use react-select helpers on the CVE compound filter, or PF typeahead helpers on global Search.
- Autocomplete with no Central data shows **No options** — still valid if the control and apply button exist. Inventing a CVE ID is worse than an empty menu.
- `CompoundSearchFilter.cy.jsx` is a **component** test. Passing it is not console proof.
- Search page is permission-gated. If the masthead button is missing, check `isRouteEnabled('search')`, do not force `/main/search` and call a 403 a product bug without checking RBAC.
