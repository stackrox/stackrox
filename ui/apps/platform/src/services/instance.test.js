import { appendOrchestratorComponentsQuery, orchestratorQueryKey } from './instance';

describe('appendOrchestratorComponentsQuery', () => {
    it('should add query to incoming url if orchestrator toggle is on', () => {
        const url = '/v1/metadata';
        const newUrl = appendOrchestratorComponentsQuery(url, 'true');
        expect(newUrl).toEqual(`${url}?${orchestratorQueryKey}=true`);
    });

    it('should append to incoming url query if orchestrator toggle is on', () => {
        const url = 'v1/search/metadata/options?categories=DEPLOYMENTS';
        const newUrl = appendOrchestratorComponentsQuery(url, 'true');
        expect(newUrl).toEqual(`${url}&${orchestratorQueryKey}=true`);
    });
});
