import { ComponentType } from 'react';
import {
    GraphElement,
    ComponentFactory,
    ModelKind,
    GraphComponent,
    DefaultNode,
} from '@patternfly/react-topology';
import Edge from './DefaultEdge';
import Group from './DefaultGroup';

// @ts-expect-error TODO: raise type error issue with patternfly/react-topology team
const defaultComponentFactory: ComponentFactory = (
    kind: ModelKind,
    type: string
):
    | ComponentType<{ element: GraphElement }>
    | typeof Group
    | typeof GraphComponent
    | typeof DefaultNode
    | typeof Edge
    | typeof undefined => {
    switch (type) {
        case 'group':
            return Group;
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
