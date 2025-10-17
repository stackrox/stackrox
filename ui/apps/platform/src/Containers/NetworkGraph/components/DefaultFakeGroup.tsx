/* eslint-disable no-void */
/* eslint-disable no-cond-assign */
/* eslint-disable no-return-assign */
import { Fragment, createElement } from 'react';
import { observer } from 'mobx-react';
import { css } from '@patternfly/react-styles';
import styles from '@patternfly/react-topology/dist/js/css/topology-components';
import {
    Layer,
    LabelPosition,
    LabelBadge,
    Ellipse,
    NodeLabel,
    createSvgIdUrl,
    useCombineRefs,
    useHover,
    useSize,
    useDragNode,
} from '@patternfly/react-topology';

const DefaultFakeGroup = ({
    className,
    children,
    element,
    selected,
    onSelect,
    hover,
    label,
    secondaryLabel,
    showLabel = true,
    truncateLength,
    collapsedWidth,
    collapsedHeight,
    getCollapsedShape,
    collapsedShadowOffset = 8,
    dragNodeRef,
    dragging,
    labelPosition,
    badge,
    badgeColor,
    badgeTextColor,
    badgeBorderColor,
    badgeClassName,
    badgeLocation,
    labelIconClass,
    labelIcon,
    labelIconPadding,
}) => {
    let _a: number | undefined;
    const [hovered, hoverRef] = useHover();
    const [labelHover, labelHoverRef] = useHover();
    const dragLabelRef = useDragNode()[1];
    const [shapeSize, shapeRef] = useSize([collapsedWidth, collapsedHeight]);
    const refs = useCombineRefs(hoverRef, dragNodeRef, shapeRef);
    const isHover = hover !== undefined ? hover : hovered;
    const childCount: number = element.data.numFlows;
    const [badgeSize, badgeRef] = useSize([childCount]);
    const groupClassName = css(
        styles.topologyGroup,
        className,
        dragging && 'pf-m-dragging',
        selected && 'pf-m-selected'
    );
    const ShapeComponent = getCollapsedShape ? getCollapsedShape(element) : Ellipse;
    const filter = isHover || dragging ? createSvgIdUrl('NodeShadowsFilterId--hover') : undefined;
    return createElement(
        'g',
        { ref: labelHoverRef, onClick: onSelect, className: groupClassName },
        // eslint-disable-next-line react/no-children-prop
        createElement(Layer, {
            id: 'groups',
            children: createElement(
                'g',
                { ref: refs, onClick: onSelect },
                ShapeComponent &&
                    createElement(
                        Fragment,
                        null,
                        createElement(
                            'g',
                            { transform: `translate(${collapsedShadowOffset * 2}, 0)` },
                            createElement(ShapeComponent, {
                                className: css(styles.topologyNodeBackground, 'pf-m-disabled'),
                                element,
                                width: collapsedWidth,
                                height: collapsedHeight,
                            })
                        ),
                        createElement(
                            'g',
                            { transform: `translate(${collapsedShadowOffset}, 0)` },
                            createElement(ShapeComponent, {
                                className: css(styles.topologyNodeBackground, 'pf-m-disabled'),
                                element,
                                width: collapsedWidth,
                                height: collapsedHeight,
                            })
                        ),
                        createElement(ShapeComponent, {
                            className: css(styles.topologyNodeBackground),
                            key:
                                isHover || dragging ? 'shape-background-hover' : 'shape-background',
                            element,
                            width: collapsedWidth,
                            height: collapsedHeight,
                            filter,
                        })
                    )
            ),
        }),
        shapeSize &&
            createElement(LabelBadge, {
                className: styles.topologyGroupCollapsedBadge,
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore TS2769: No overload matches this call.
                ref: badgeRef,
                x: shapeSize.width - 8,
                y:
                    (shapeSize.width -
                        ((_a =
                            badgeSize === null || badgeSize === void 0
                                ? void 0
                                : badgeSize.height) !== null && _a !== void 0
                            ? _a
                            : 0)) /
                    2,
                badge: `${childCount}`,
                badgeColor,
                badgeTextColor,
                badgeBorderColor,
            }),
        showLabel &&
            createElement(
                NodeLabel,
                {
                    className: styles.topologyGroupLabel,
                    x:
                        labelPosition === LabelPosition.right
                            ? collapsedWidth + 8
                            : collapsedWidth / 2,
                    y:
                        labelPosition === LabelPosition.right
                            ? collapsedHeight / 2
                            : collapsedHeight + 6,
                    paddingX: 8,
                    paddingY: 5,
                    dragRef: dragNodeRef ? dragLabelRef : undefined,
                    status: element.getNodeStatus(),
                    secondaryLabel,
                    truncateLength,
                    badge,
                    badgeColor,
                    badgeTextColor,
                    badgeBorderColor,
                    badgeClassName,
                    badgeLocation,
                    labelIconClass,
                    labelIcon,
                    labelIconPadding,
                    hover: isHover || labelHover,
                },
                label || element.getLabel()
            ),
        children
    );
};
export default observer(DefaultFakeGroup);
