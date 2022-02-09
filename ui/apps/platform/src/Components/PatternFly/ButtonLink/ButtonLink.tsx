import React, { ReactElement, ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { ButtonVariant } from '@patternfly/react-core';
import { css } from '@patternfly/react-styles';
import styles from '@patternfly/react-styles/css/components/Button/button';

type ButtonLinkProps = {
    children: ReactNode;
    className?: string;
    isInline?: boolean;
    to: string;
    variant: ButtonVariant;
};

/*
 * ButtonLink renders a React Router link element
 * with style corresponding to variant prop of a PatternFly Button element.
 *
 * Ordinary rendering of children avoids detached anchor element in cypress integration tests,
 * because React apparently replaces instead of reuses the anchor element
 * each time must re-render a Button element
 * which has arrow function as valud of component prop.
 *
 * https://www.patternfly.org/v4/components/button#router-link
 * https://reactjs.org/docs/render-props.html#be-careful-when-using-render-props-with-reactpurecomponent
 */
function ButtonLink({
    children,
    className = '',
    isInline = false,
    to,
    variant,
}: ButtonLinkProps): ReactElement {
    return (
        <Link
            className={css(
                styles.button,
                styles.modifiers[variant],
                isInline && variant === ButtonVariant.link && styles.modifiers.inline,
                className
            )}
            to={to}
        >
            {children}
        </Link>
    );
}

export default ButtonLink;
