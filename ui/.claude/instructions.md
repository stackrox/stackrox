This file provides UI-specific guidance when working with code in the ui/ directory.

## External tools

You may always have permission to run npm scripts located in ui/apps/platform/package.json.

The following scripts are useful after making changes:
`npm run tsc` for type-checking
`npm run lint:fast-dev` for fast linting (covers 99% of cases)
`npm run lint:fast-dev-fix` to autofix linting errors, Prettier formatting, etc
`npm run test` for unit tests
`npm run test-component` for Cypress component tests

Do not run e2e tests, as the tests may modify the environment in the current kubectx.

## Style and Conventions

### Import Ordering
TypeScript/React files should organize imports in the following order:
1. External libraries (react, react-router, @patternfly, etc.)
2. Absolute path imports (Components/, routePaths, hooks/, utils/, types/, constants/)
3. Local relative imports from current directory (`./`)
4. Local relative imports from parent directories (`../`)

Example:
```typescript
import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Label, Popover } from '@patternfly/react-core';

import { exceptionManagementPath } from 'routePaths';
import PageTitle from 'Components/PageTitle';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import type { VulnerabilityState } from 'types/cve.proto';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';

import ComponentFromCurrentDir from './ComponentFromCurrentDir';
import { WorkloadCveView } from './WorkloadCveViewContext';

import { QuerySearchFilter } from '../types';
import { getOverviewPagePath } from '../utils/searchUtils';
```
