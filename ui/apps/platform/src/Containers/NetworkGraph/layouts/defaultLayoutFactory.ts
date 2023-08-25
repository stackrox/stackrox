import {
    Graph,
    Layout,
    LayoutFactory,
    ForceLayout,
    ColaLayout,
    ConcentricLayout,
    DagreLayout,
    GridLayout,
    BreadthFirstLayout,
} from '@patternfly/react-topology';
import { ColaGroupsLayout } from '@patternfly/react-topology/dist/esm/layouts/ColaGroupsLayout';

const defaultLayoutFactory: LayoutFactory = (type: string, graph: Graph): Layout | undefined => {
    switch (type) {
        case 'BreadthFirst':
            return new BreadthFirstLayout(graph);
        case 'Cola':
            return new ColaLayout(graph);
        case 'ColaNoForce':
            return new ColaLayout(graph, { layoutOnDrag: false });
        case 'Concentric':
            return new ConcentricLayout(graph);
        case 'Dagre':
            return new DagreLayout(graph);
        case 'Force':
            return new ForceLayout(graph);
        case 'Grid':
            return new GridLayout(graph);
        case 'ColaGroups':
            return new ColaGroupsLayout(graph, { layoutOnDrag: false });
        default:
            return new ColaLayout(graph, { layoutOnDrag: false });
    }
};

export default defaultLayoutFactory;
