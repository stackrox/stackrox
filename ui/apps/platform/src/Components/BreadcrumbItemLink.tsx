import React from 'react';
import { BreadcrumbItem, BreadcrumbItemProps } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

export type BreadcrumbItemLinkProps = {
    children: React.ReactNode;
    to: string;
} & BreadcrumbItemProps;

function BreadcrumbItemLink({
    children,
    to,
    ...rest
}: BreadcrumbItemLinkProps): React.ReactElement {
    function render({ className, ariaCurrent }) {
        return (
            <Link className={className} aria-current={ariaCurrent} to={to}>
                {children}
            </Link>
        );
    }
    return <BreadcrumbItem {...rest} render={render} />;
}

export default BreadcrumbItemLink;
