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
    onHoveredRowComponentRender?: (row) => ReactNode;
};

const tableRowClassName = 'relative border-b';
const baseTableRowClassName = `${tableRowClassName} border-base-300`;
const alertTableRowClassName = `${tableRowClassName} border-alert-300 bg-alert-200 text-alert-800`;

function onFocus(): number {
    return 0;
}

function TableRow({
    type,
    row,
    children,
    onHoveredRowComponentRender,
}: TableRowProps): ReactElement {
    const [isHovered, setIsHovered] = useState(false);

    const { key } = row.getRowProps();
    const className = type === 'alert' ? alertTableRowClassName : baseTableRowClassName;
    const shouldShowHoveredComponent = onHoveredRowComponentRender && !row.isGrouped && isHovered;

    const hoveredComponent = shouldShowHoveredComponent && (
        <td className="absolute right-0 transform -translate-x-2 translate-y-1 mr-2">
            {onHoveredRowComponentRender && onHoveredRowComponentRender(row)}
        </td>
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
        >
            {children}
            {hoveredComponent}
        </tr>
    );
}

export default TableRow;
