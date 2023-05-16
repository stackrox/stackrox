import { AdvancedFlowsFilterType, FilterValue } from './types';

function isValidPort(port: string): boolean {
    const portNum = parseInt(port, 10);
    if (Number.isNaN(portNum)) {
        return false;
    }
    return portNum >= 1 && portNum <= 65535;
}

// This function is used to convert our filters data structure to an array of strings
// which the Select component expects
export function filtersToSelections(filters: AdvancedFlowsFilterType): FilterValue[] {
    return Object.values(filters).reduce((acc, curr) => {
        return [...acc, ...Object.values(curr)];
    }, []);
}

// This function is used to convert the selections array of strings, which the Select component
// expects, into our filters data structure
export function selectionsToFilters(selections: string[]): AdvancedFlowsFilterType {
    const filters: AdvancedFlowsFilterType = {
        directionality: [],
        protocols: [],
        ports: [],
    };
    selections.forEach((selection) => {
        if (selection === 'ingress' || selection === 'egress') {
            filters.directionality.push(selection);
        } else if (selection === 'L4_PROTOCOL_TCP' || selection === 'L4_PROTOCOL_UDP') {
            filters.protocols.push(selection);
        } else if (isValidPort(selection)) {
            filters.ports.push(selection);
        } else {
            // do nothing
        }
    });
    return filters;
}
