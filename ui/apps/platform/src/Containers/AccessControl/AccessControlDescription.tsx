import React, { ReactElement, ReactNode } from 'react';

export type AccessControlDescriptionProps = {
    children: ReactNode;
};

/*
 * Render description following AccessControlNav and preceding Title h2 in list or form element.
 */
function AccessControlDescription({ children }: AccessControlDescriptionProps): ReactElement {
    return <div className="pf-u-font-size-sm pf-u-pt-sm">{children}</div>;
}

export default AccessControlDescription;
