import type { ReactElement, ReactNode } from 'react';

export type AccessControlDescriptionProps = {
    children: ReactNode;
};

/*
 * Render description following AccessControlNav and preceding Title h2 in list or h1 in form element.
 */
function AccessControlDescription({ children }: AccessControlDescriptionProps): ReactElement {
    return <div className="pf-v5-u-font-size-sm pf-v5-u-pt-sm">{children}</div>;
}

export default AccessControlDescription;
