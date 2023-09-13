import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import WorkflowEntity from 'utils/WorkflowEntity';
import { WorkflowState } from 'utils/WorkflowState';

import { getCveTableColumns } from './VulnMgmtListCves';
import { getFilteredCVEColumns, parseCveNamesFromIds } from './ListCVEs.utils';

function mockIsFeatureFlagEnabled(flag) {
    if (flag === 'ROX_ACTIVE_VULN_MGMT') {
        return true;
    }
    return false;
}

describe('ListCVEs.utils', () => {
    describe('getFilteredCVEColumns', () => {
        it('should return all the cve columns when in a context that allows them', () => {
            const stateStack = [
                new WorkflowEntity(entityTypes.IMAGE_COMPONENT),
                new WorkflowEntity(entityTypes.IMAGE_CVE),
            ];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getCveTableColumns(workflowState, mockIsFeatureFlagEnabled);

            const filteredColumns = getFilteredCVEColumns(
                tableColumns,
                workflowState,
                mockIsFeatureFlagEnabled
            );

            expect(filteredColumns).toEqual(tableColumns);
        });

        it('should remove the fixed in columns when in CVE main list context', () => {
            const stateStack = [new WorkflowEntity(entityTypes.IMAGE_CVE)];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getCveTableColumns(workflowState, mockIsFeatureFlagEnabled);

            const filteredColumns = getFilteredCVEColumns(
                tableColumns,
                workflowState,
                mockIsFeatureFlagEnabled
            );

            const locationColumnPresent = filteredColumns.find(
                (col) => col.accessor === 'fixedByVersion'
            );
            expect(locationColumnPresent).toBeUndefined();
        });

        it('should remove the fixed in column when in CVE sublist of Deployment single context', () => {
            const stateStack = [
                new WorkflowEntity(entityTypes.DEPLOYMENT, 'abcd-ef09'),
                new WorkflowEntity(entityTypes.IMAGE_CVE),
            ];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getCveTableColumns(workflowState, mockIsFeatureFlagEnabled);

            const filteredColumns = getFilteredCVEColumns(
                tableColumns,
                workflowState,
                mockIsFeatureFlagEnabled
            );

            const locationColumnPresent = filteredColumns.find(
                (col) => col.accessor === 'fixedByVersion'
            );
            expect(locationColumnPresent).toBeUndefined();
        });

        it('should show the fixed in column when in CVE sublist of Component single context', () => {
            const stateStack = [
                new WorkflowEntity(entityTypes.IMAGE_COMPONENT, 'abcd-ef09'),
                new WorkflowEntity(entityTypes.IMAGE_CVE),
            ];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getCveTableColumns(workflowState, mockIsFeatureFlagEnabled);

            const filteredColumns = getFilteredCVEColumns(
                tableColumns,
                workflowState,
                mockIsFeatureFlagEnabled
            );

            expect(filteredColumns).toEqual(tableColumns);
        });
    });

    describe('parseCveNamesFromIds', () => {
        it('should return an empty array when passed an empty array', () => {
            const selectedCveIds = [];

            const parseCveNames = parseCveNamesFromIds(selectedCveIds);

            expect(parseCveNames).toEqual([]);
        });

        it('should return just the first CVE name parts of a list of CVE IDs', () => {
            const selectedCveIds = ['CVE-2005-2541#debian:12', 'CVE-2014-7187#debian:8'];

            const parseCveNames = parseCveNamesFromIds(selectedCveIds);

            expect(parseCveNames).toEqual(['CVE-2005-2541', 'CVE-2014-7187']);
        });

        it('should return a deduped list of first CVE name parts, when multiple CVE IDs start the same', () => {
            const selectedCveIds = [
                'CVE-2005-2541#debian:11',
                'CVE-2005-2541#debian:10',
                'CVE-2005-2541#debian:12',
                'CVE-2014-7187#debian:8',
                'CVE-2004-0971#debian:9',
                'CVE-2004-0971#unknown',
            ];

            const parseCveNames = parseCveNamesFromIds(selectedCveIds);

            expect(parseCveNames).toEqual(['CVE-2005-2541', 'CVE-2014-7187', 'CVE-2004-0971']);
        });
    });
});
