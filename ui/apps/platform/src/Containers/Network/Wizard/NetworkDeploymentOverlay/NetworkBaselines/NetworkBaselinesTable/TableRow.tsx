import React, { ReactElement, ReactNode, useState } from 'react';

export type TableRowProps = {
    type: 'alert' | null;
    row: {
        getRowProps: () => {
            key: string;
        };
        isGrouped: boolean;
    };
    children: ReactNode;
    HoveredRowComponent?: ReactNode;
    HoveredGroupedRowComponent?: ReactNode;
    GroupedRowComponent?: ReactNode;
};

const tableRowClassName = 'relative border-b';
const baseTableRowClassName = `${tableRowClassName} border-base-300 bg-base-100`;
const alertTableRowClassName = `${tableRowClassName} border-alert-300 bg-alert-200 text-alert-800`;

function onFocus(): number {
    return 0;
}

function TableRowOverlay({ children }): ReactElement {
    return (
        <td className="flex overflow-visible w-0">
            <div className="-translate-x-full transform translate-y-1 whitespace-no-wrap z-0">
                {children}
            </div>
        </td>
    );
}

function TableRow({
    type,
    row,
    children,
    HoveredRowComponent = null,
    HoveredGroupedRowComponent = null,
    GroupedRowComponent = null,
}: TableRowProps): ReactElement {
    const [isHovered, setIsHovered] = useState(false);

    const { key } = row.getRowProps();
    const className = type === 'alert' ? alertTableRowClassName : baseTableRowClassName;
    const showGroupedRowComponent = row.isGrouped;
    const showHoveredRowComponent = !showGroupedRowComponent && isHovered;
    const showHoveredGroupedRowComponent = showGroupedRowComponent && isHovered;

    const hoveredRowComponent = showHoveredRowComponent && HoveredRowComponent && (
        <TableRowOverlay>{HoveredRowComponent}</TableRowOverlay>
    );

    const groupedRowComponent = showGroupedRowComponent && GroupedRowComponent && (
        <TableRowOverlay>{GroupedRowComponent}</TableRowOverlay>
    );

    const hoveredGroupedRowComponent = showHoveredGroupedRowComponent &&
        HoveredGroupedRowComponent && (
            <TableRowOverlay>{HoveredGroupedRowComponent}</TableRowOverlay>
        );

    function onMouseEnter(): void {
        setIsHovered(true);
    }

    function onMouseLeave(): void {
        setIsHovered(false);
    }

    return (
        <tr
            key={key}
            className={className}
            onMouseEnter={onMouseEnter}
            onMouseLeave={onMouseLeave}
            onFocus={onFocus}
            data-testid={Array.isArray(children) ? 'data-row' : 'subhead-row'}
        >
            {children}
            {hoveredRowComponent}
            {groupedRowComponent}
            {hoveredGroupedRowComponent}
        </tr>
    );
}

export default TableRow;
