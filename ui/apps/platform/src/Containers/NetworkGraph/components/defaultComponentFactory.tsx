import { ComponentType } from 'react';
import {
    GraphElement,
    ComponentFactory,
    ModelKind,
    GraphComponent,
    DefaultNode,
} from '@patternfly/react-topology';
import Edge from './DefaultEdge';
import MultiEdge from './MultiEdge';
import Group from './DefaultGroup';
import GroupHull from './GroupHull';

// @ts-expect-error TODO: raise type error issue with patternfly/react-topology team
const defaultComponentFactory: ComponentFactory = (
    kind: ModelKind,
    type: string
):
    | ComponentType<{ element: GraphElement }>
    | typeof MultiEdge
    | typeof Group
    | typeof GroupHull
    | typeof GraphComponent
    | typeof DefaultNode
    | typeof Edge
    | typeof undefined => {
    switch (type) {
        case 'multi-edge':
            return MultiEdge;
        case 'group':
            return Group;
        case 'group-hull':
            return GroupHull;
        default:
            switch (kind) {
                case ModelKind.graph:
                    return GraphComponent;
                case ModelKind.node:
                    return DefaultNode;
                case ModelKind.edge:
                    return Edge;
                default:
                    return undefined;
            }
    }
};

export default defaultComponentFactory;
