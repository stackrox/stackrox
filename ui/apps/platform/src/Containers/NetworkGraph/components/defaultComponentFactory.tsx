import { ComponentType } from 'react';
import {
    GraphElement,
    ComponentFactory,
    ModelKind,
    GraphComponent,
    DefaultNode,
    DefaultEdge,
    DefaultGroup,
} from '@patternfly/react-topology';

// @ts-expect-error TODO: raise type error issue with patternfly/react-topology team
const defaultComponentFactory: ComponentFactory = (
    kind: ModelKind,
    type: string
):
    | ComponentType<{ element: GraphElement }>
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
                case ModelKind.graph:
                    return GraphComponent;
                case ModelKind.node:
                    return DefaultNode;
                case ModelKind.edge:
                    return DefaultEdge;
                default:
                    return undefined;
            }
    }
};

export default defaultComponentFactory;
