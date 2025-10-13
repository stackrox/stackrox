import React, { useMemo } from 'react';
import type { ComponentClass, FunctionComponent, PropsWithChildren, ReactNode } from 'react';
import { DefaultGroup, observer, ScaleDetailsLevel } from '@patternfly/react-topology';
import type {
    Node,
    ShapeProps,
    WithDragNodeProps,
    WithSelectionProps,
} from '@patternfly/react-topology';
import AlternateIcon from '@patternfly/react-icons/dist/esm/icons/regions-icon';
import DefaultIcon from '@patternfly/react-icons/dist/esm/icons/builder-image-icon';
import useDetailsLevel from '@patternfly/react-topology/dist/esm/hooks/useDetailsLevel';
import type { SVGIconProps } from '@patternfly/react-icons/dist/js/createIcon';
import type { CustomGroupNodeData } from '../types/topology.type';

const ICON_PADDING = 20;

export enum DataTypes {
    Default,
    Alternate,
}

type StyleGroupProps = {
    element: Node;
    collapsible: boolean;
    collapsedWidth?: number;
    collapsedHeight?: number;
    onCollapseChange?: (group: Node, collapsed: boolean) => void;
    getCollapsedShape?: (node: Node) => FunctionComponent<PropsWithChildren<ShapeProps>>;
    collapsedShadowOffset?: number; // defaults to 10
} & WithDragNodeProps &
    WithSelectionProps;

const StyleGroup: FunctionComponent<PropsWithChildren<StyleGroupProps>> = ({
    element,
    collapsedWidth = 75,
    collapsedHeight = 75,
    ...rest
}) => {
    const data = element.getData();
    const detailsLevel = useDetailsLevel();

    const getTypeIcon = (dataType?: DataTypes): ComponentClass<SVGIconProps> => {
        switch (dataType) {
            case DataTypes.Alternate:
                return AlternateIcon;
            default:
                return DefaultIcon;
        }
    };

    const renderIcon = (): ReactNode => {
        const iconSize = Math.min(collapsedWidth, collapsedHeight) - ICON_PADDING * 2;
        const Component = getTypeIcon(data.dataType);

        return (
            <g
                transform={`translate(${(collapsedWidth - iconSize) / 2}, ${
                    (collapsedHeight - iconSize) / 2
                })`}
            >
                <Component style={{ color: '#393F44' }} width={iconSize} height={iconSize} />
            </g>
        );
    };

    const passedData = useMemo(() => {
        const newData = { ...data };
        Object.keys(newData).forEach((key) => {
            if (newData[key] === undefined) {
                delete newData[key];
            }
        });
        // look into using `useMemo<CustomGroupNodeData>` instead of `as CustomGroupNodeData`
        // https://www.freecodecamp.org/news/react-typescript-how-to-set-up-types-on-hooks/#set-types-on-usememo
        return newData as CustomGroupNodeData;
    }, [data]);

    let className = '';

    if (passedData.type === 'NAMESPACE') {
        className = `${className} ${
            passedData.isFilteredNamespace ? 'filtered-namespace' : 'related-namespace'
        }`.trim();
    }

    // @TODO: If multiple classes need to be stringed together, then we need a more systematic way to generate those here
    className = `${className} ${passedData?.isFadedOut ? 'pf-topology-node-faded' : ''}`.trim();

    return (
        /*
        (dv 2024-05-01)
        Upgrading to React types 18 causes a type error below as React 18 no longer includes `children`
        as a prop of `React.FC` by default

        @ts-expect-error DefaultGroup does not expect children as a prop */
        <DefaultGroup
            element={element}
            collapsedWidth={collapsedWidth}
            collapsedHeight={collapsedHeight}
            showLabel={detailsLevel === ScaleDetailsLevel.high}
            className={className}
            {...rest}
            {...passedData}
        >
            {element.isCollapsed() ? renderIcon() : null}
        </DefaultGroup>
    );
};

export default observer(StyleGroup);
