import { RestSearchOption } from 'services/searchOptionsToQuery';

export const orchestratorComponentsOption: RestSearchOption[] = [
    {
        value: 'Orchestrator Component:',
        type: 'categoryOption',
    },
    {
        value: 'false',
    },
];

export const ORCHESTRATOR_COMPONENTS_KEY = 'showOrchestratorComponents';
