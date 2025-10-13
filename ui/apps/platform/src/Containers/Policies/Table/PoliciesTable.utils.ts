import { sortSeverity, sortAsciiCaseInsensitive, sortValueByLength } from 'sorters/sorters';
import { ListPolicy } from 'types/policy.proto';
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
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortValueByLength(a.notifiers, b.notifiers),
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
