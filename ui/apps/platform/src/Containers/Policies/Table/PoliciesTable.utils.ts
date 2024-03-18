import { sortSeverity, sortAsciiCaseInsensitive, sortValueByLength } from 'sorters/sorters';
import { ListPolicy } from 'types/policy.proto';

export const defaultPolicyLabel = 'System';
export const userPolicyLabel = 'User';

function sortPolicyOrigin(a, b) {
    const aOrigin = a ? defaultPolicyLabel : userPolicyLabel;
    const bOrigin = b ? defaultPolicyLabel : userPolicyLabel;

    return sortAsciiCaseInsensitive(aOrigin, bOrigin);
}

export const columns = [
    {
        Header: 'Policy',
        accessor: 'name',
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortAsciiCaseInsensitive(a.name, b.name),
        width: 30 as const,
    },
    {
        Header: 'Status',
        accessor: 'disabled',
    },
    {
        Header: 'Origin',
        accessor: 'isDefault',
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortPolicyOrigin(a.isDefault, b.isDefault),
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
