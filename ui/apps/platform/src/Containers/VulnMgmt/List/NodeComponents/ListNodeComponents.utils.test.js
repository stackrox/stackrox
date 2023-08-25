import entityTypes from 'constants/entityTypes';
import { componentSortFields } from 'constants/sortFields';
import useCases from 'constants/useCaseTypes';
import WorkflowEntity from 'utils/WorkflowEntity';
import { WorkflowState } from 'utils/WorkflowState';

import { getFilteredComponentColumns } from './ListNodeComponents.utils';

describe('ListNodeComponents.utils', () => {
    describe('getFilteredComponentColumns', () => {
        it('should return all the components columns when in a context that allows them', () => {
            const stateStack = [
                new WorkflowEntity(entityTypes.IMAGE, 'abcd-ef09'),
                new WorkflowEntity(entityTypes.COMPONENT),
            ];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getComponentTableColumns(workflowState);

            const filteredColumns = getFilteredComponentColumns(tableColumns, workflowState);

            expect(filteredColumns).toEqual(tableColumns);
        });

        it('should remove the source and location columns when in Components main list context', () => {
            const stateStack = [new WorkflowEntity(entityTypes.COMPONENT)];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getComponentTableColumns(workflowState);

            const filteredColumns = getFilteredComponentColumns(tableColumns, workflowState);

            const sourceColumnPresent = filteredColumns.find((col) => col.accessor === 'source');
            expect(sourceColumnPresent).toBeUndefined();
            const locationColumnPresent = filteredColumns.find(
                (col) => col.accessor === 'location'
            );
            expect(locationColumnPresent).toBeUndefined();
        });

        it('should remove the source and location columns when in Components sublist of CVE single context', () => {
            const stateStack = [
                new WorkflowEntity(entityTypes.CVE, 'abcd-ef09'),
                new WorkflowEntity(entityTypes.COMPONENT),
            ];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getComponentTableColumns(workflowState);

            const filteredColumns = getFilteredComponentColumns(tableColumns, workflowState);

            const sourceColumnPresent = filteredColumns.find((col) => col.accessor === 'source');
            expect(sourceColumnPresent).toBeUndefined();
            const locationColumnPresent = filteredColumns.find(
                (col) => col.accessor === 'location'
            );
            expect(locationColumnPresent).toBeUndefined();
        });

        it('should remove the source and location columns when in Components sublist of Deployment single context', () => {
            const stateStack = [
                new WorkflowEntity(entityTypes.DEPLOYMENT, 'abcd-ef09'),
                new WorkflowEntity(entityTypes.COMPONENT),
            ];
            const workflowState = new WorkflowState(useCases.VULN_MANAGEMENT, stateStack);
            const tableColumns = getComponentTableColumns(workflowState);

            const filteredColumns = getFilteredComponentColumns(tableColumns, workflowState);

            const sourceColumnPresent = filteredColumns.find((col) => col.accessor === 'source');
            expect(sourceColumnPresent).toBeUndefined();
            const locationColumnPresent = filteredColumns.find(
                (col) => col.accessor === 'location'
            );
            expect(locationColumnPresent).toBeUndefined();
        });
    });
});

function getComponentTableColumns(workflowState) {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: `Component`,
            headerClassName: `w-1/4`,
            className: `w-1/4`,
            Cell: ({ original }) => {
                const { version, name } = original;
                return `${name} ${version}`;
            },
            accessor: 'name',
            sortField: componentSortFields.COMPONENT,
        },
        {
            Header: `CVEs`,
            entityType: entityTypes.CVE,
            headerClassName: `w-1/8`,
            className: `w-1/8`,
            Cell: ({ original }) => {
                const { vulnCounter, id } = original;
                if (!vulnCounter || vulnCounter.all.total === 0) {
                    return 'No CVEs';
                }

                const newState = workflowState.pushListItem(id).pushList(entityTypes.CVE);
                const url = newState.toUrl();
                const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                return `${vulnCounter}${url}${fixableUrl}`;
            },
            accessor: 'vulnCounter.all.total',
            sortField: componentSortFields.CVE_COUNT,
        },
        {
            Header: `Top CVSS`,
            headerClassName: `w-1/10 text-center`,
            className: `w-1/10`,
            Cell: ({ original }) => {
                const { topVuln } = original;
                if (!topVuln) {
                    return '–';
                }
                const { cvss, scoreVersion } = topVuln;
                return `${cvss} ${scoreVersion}`;
            },
            accessor: 'topVuln.cvss',
            sortField: componentSortFields.TOP_CVSS,
        },
        {
            Header: `Source`,
            headerClassName: `w-1/8`,
            className: `w-1/8`,
            accessor: 'source',
            // @TODO uncomment once source is sortable on backend
            // sortField: componentSortFields.SOURCE
        },
        {
            Header: `Location`,
            headerClassName: `w-1/8`,
            className: `w-1/8`,
            accessor: 'location',
            sortable: false,
        },
        {
            Header: `Images`,
            entityType: entityTypes.IMAGE,
            headerClassName: `w-1/8`,
            className: `w-1/8`,
            accessor: 'imageCount',
            Cell: ({ original }) => original.imageCount,
            // TODO: restore sorting on this field, see https://issues.redhat.com/browse/ROX-12548 for context
            // sortField: componentSortFields.IMAGES,
            sortable: false,
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/8`,
            className: `w-1/8`,
            accessor: 'deploymentCount',
            Cell: ({ original }) => original.deploymentCount,
            sortField: componentSortFields.DEPLOYMENTS,
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/10`,
            className: `w-1/10`,
            accessor: 'priority',
            sortField: componentSortFields.PRIORITY,
        },
    ];

    return [...tableColumns];
}
