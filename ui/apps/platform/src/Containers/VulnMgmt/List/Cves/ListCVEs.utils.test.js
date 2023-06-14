import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import WorkflowEntity from 'utils/WorkflowEntity';
import { WorkflowState } from 'utils/WorkflowState';

import { getCveTableColumns } from './VulnMgmtListCves';
import { getFilteredCVEColumns } from './ListCVEs.utils';

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
});
