import type { ComponentType, PropsWithChildren } from 'react';
import { DefaultEdge, DefaultGroup, DefaultNode, GraphComponent } from '@patternfly/react-topology';
import type { ComponentFactory, GraphElement } from '@patternfly/react-topology';

import { ensureExhaustive } from 'utils/type.utils';

type CustomModelKind = 'node' | 'graph' | 'edge' | 'fakeGroup';

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
                case 'graph':
                    return GraphComponent;
                case 'node':
                    return DefaultNode;
                case 'edge':
                    return DefaultEdge;
                case 'fakeGroup':
                    return DefaultNode;
                default:
                    return ensureExhaustive(kind);
            }
    }
};

export default defaultComponentFactory;
