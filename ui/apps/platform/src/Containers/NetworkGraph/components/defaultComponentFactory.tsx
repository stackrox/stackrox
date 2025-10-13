import type { ComponentType, PropsWithChildren } from 'react';
import { GraphComponent, DefaultNode, DefaultEdge, DefaultGroup } from '@patternfly/react-topology';
import type { ComponentFactory, GraphElement } from '@patternfly/react-topology';

enum CustomModelKind {
    node = 'node',
    graph = 'graph',
    edge = 'edge',
    fakeGroup = 'fakeGroup',
}

// @ts-expect-error TODO: raise type error issue with patternfly/react-topology team
const defaultComponentFactory: ComponentFactory = (
    kind: CustomModelKind,
    type: string
):
    | ComponentType<PropsWithChildren<{ element: GraphElement }>>
    | typeof DefaultGroup
    | typeof GraphComponent
    | typeof DefaultNode
    | typeof DefaultEdge
    | typeof undefined => {
    switch (type) {
        case 'group':
            return DefaultGroup;
        default:
            switch (kind) {
                case CustomModelKind.graph:
                    return GraphComponent;
                case CustomModelKind.node:
                    return DefaultNode;
                case CustomModelKind.edge:
                    return DefaultEdge;
                case CustomModelKind.fakeGroup:
                    return DefaultNode;
                default:
                    return undefined;
            }
    }
};

export default defaultComponentFactory;
