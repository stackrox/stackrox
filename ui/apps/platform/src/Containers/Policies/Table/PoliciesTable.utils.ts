import { sortAsciiCaseInsensitive, sortSeverity } from 'sorters/sorters';
import type { ListPolicy } from 'types/policy.proto';
import { getPolicyOriginLabel } from '../policies.utils';

export const columns = [
    {
        Header: 'Policy',
        accessor: 'name',
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortAsciiCaseInsensitive(a.name, b.name),
    },
    {
        Header: 'Status',
        accessor: 'disabled',
    },
    {
        Header: 'Origin',
        accessor: 'isDefault',
        sortMethod: (a: ListPolicy, b: ListPolicy) =>
            sortAsciiCaseInsensitive(getPolicyOriginLabel(a), getPolicyOriginLabel(b)),
    },
    {
        Header: 'Notifiers',
        accessor: 'notifiers',
        sortMethod: (a: ListPolicy, b: ListPolicy) => {
            const aCount =
                a.notifiers.length + (a.notifierToCollectionMappings?.length ?? 0);
            const bCount =
                b.notifiers.length + (b.notifierToCollectionMappings?.length ?? 0);
            return aCount - bCount;
        },
    },
    {
        Header: 'Severity',
        accessor: 'severity',
        sortMethod: (a: ListPolicy, b: ListPolicy) => -sortSeverity(a.severity, b.severity),
    },
    {
        Header: 'Lifecycle',
        accessor: 'lifecycleStages',
    },
];
