import { uniq } from 'lodash';
import { Flow } from '../types';

export function getAllUniquePorts(flows: Flow[]) {
    const allPorts = flows.reduce((acc, curr) => {
        if (curr.children.length) {
            return [...acc, ...curr.children.map((child) => child.port)];
        }
        return [...acc, curr.port];
    }, [] as string[]);
    const allUniquePorts = uniq(allPorts);
    return allUniquePorts;
}
