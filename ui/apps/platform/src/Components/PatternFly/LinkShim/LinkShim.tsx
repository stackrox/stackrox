import React, { AnchorHTMLAttributes, ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';

/*
 * Given href prop, return React Router Link element with to prop.
 *
 * To avoid "element is detached from the DOM" cypress error for PatternFly Button element:
 *
 * Replace render props idiom which replaces anchor element because arrow function is different for every render:
 * <Button variant={variant} component={(...props) => <Link {...props} to={href} />}>
 *
 * With shim idiom which reuses anchor element for every render:
 * <Button variant={variant} component={LinkShim} href={href}>
 * just as it would be component={Link} if Link element had href prop instead of to prop.
 */
// Component props have inconsistent name because AnchorHTMLAttributes is from react.
/* eslint-disable generic/react-props-name */
function LinkShim({
    children,
    href,
    ...rest
}: AnchorHTMLAttributes<HTMLAnchorElement>): ReactElement {
    if (!href) {
        return <></>;
    }
    return (
        <Link {...rest} to={href}>
            {children}
        </Link>
    );
}
/* eslint-enable generic/react-props-name */

export default LinkShim;
