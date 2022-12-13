import { uniq } from 'lodash';
import { Flow } from '../types/flow.type';

export function getAllUniquePorts(flows: Flow[]): string[] {
    const allPorts = flows.reduce((acc, curr) => {
        if (curr.children && curr.children.length) {
            return [...acc, ...curr.children.map((child) => child.port)];
        }
        return [...acc, curr.port];
    }, [] as string[]);
    const allUniquePorts = uniq(allPorts);
    return allUniquePorts;
}

export function getNumFlows(flows: Flow[]): number {
    const numFlows = flows.reduce((acc, curr) => {
        // if there are no children then it counts as 1 flow
        return acc + (curr.children && curr.children.length ? curr.children.length : 1);
    }, 0);
    return numFlows;
}
