// Support file for verify-ui runs: the repo support file plus the proof checks from
// evidence.js. Set as supportFile by templates/cypress.config.js.
import '../../../../apps/platform/cypress/support/e2e';
import { installEvidenceHooks } from './evidence';

installEvidenceHooks();
