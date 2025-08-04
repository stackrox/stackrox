/* eslint-disable @typescript-eslint/no-unsafe-return */
import React, { useMemo } from 'react';
import type { FunctionComponent, PropsWithChildren } from 'react';
import { observer } from 'mobx-react';
import { Edge, DefaultEdge } from '@patternfly/react-topology';

type StyleEdgeProps = {
    element: Edge;
};

const StyleEdge: FunctionComponent<PropsWithChildren<StyleEdgeProps>> = ({ element, ...rest }) => {
    const data = element.getData();

    const passedData = useMemo(() => {
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
