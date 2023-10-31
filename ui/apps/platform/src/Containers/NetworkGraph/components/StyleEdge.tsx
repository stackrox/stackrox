/* eslint-disable @typescript-eslint/no-unsafe-return */
import * as React from 'react';
import { observer } from 'mobx-react';
import { Edge, DefaultEdge } from '@patternfly/react-topology';

type StyleEdgeProps = {
    element: Edge;
};

const StyleEdge: React.FunctionComponent<React.PropsWithChildren<StyleEdgeProps>> = ({ element, ...rest }) => {
    const data = element.getData();

    const passedData = React.useMemo(() => {
        const newData = { ...data };
        Object.keys(newData).forEach((key) => {
            if (newData[key] === undefined) {
                delete newData[key];
            }
        });
        return newData;
    }, [data]);

    return <DefaultEdge element={element} {...rest} {...passedData} />;
};

export default observer(StyleEdge);
