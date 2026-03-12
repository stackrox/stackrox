import type { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';
import { getSearchFilterConfig } from './ViolationsTableSearchFilter';

function getEntityNames(view: FilteredWorkflowView): string[] {
    return getSearchFilterConfig(view).map((entity) => entity.displayName);
}

describe('getSearchFilterConfig', () => {
    it('should return only application-relevant entities for "Applications view"', () => {
        expect(getEntityNames('Applications view')).toEqual([
            'Cluster',
            'Deployment',
            'Namespace',
            'Policy',
            'Policy violation',
        ]);
    });

    it('should return only platform-relevant entities for "Platform view"', () => {
        expect(getEntityNames('Platform view')).toEqual([
            'Cluster',
            'Deployment',
            'Namespace',
            'Policy',
            'Policy violation',
        ]);
    });

    it('should return only node-relevant entities for "Node view"', () => {
        expect(getEntityNames('Node view')).toEqual([
            'Cluster',
            'Policy',
            'Policy violation',
            'Node',
        ]);
    });

    it('should return all entities for "Full view"', () => {
        expect(getEntityNames('Full view')).toEqual([
            'Cluster',
            'Deployment',
            'Namespace',
            'Policy',
            'Policy violation',
            'Node',
            'Resource',
        ]);
    });

    it('should always include "Policy" as a valid entity for all views', () => {
        const views: FilteredWorkflowView[] = [
            'Applications view',
            'Platform view',
            'Node view',
            'Full view',
        ];
        views.forEach((view) => {
            expect(getEntityNames(view)).toContain('Policy');
        });
    });
});
