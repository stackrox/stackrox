import React, { ReactElement, ReactNode } from 'react';

export type AccessControlDescriptionProps = {
    children: ReactNode;
};

/*
 * Render description following AccessControlNav and preceding Title h2 in list or form element.
 */
function AccessControlDescription({ children }: AccessControlDescriptionProps): ReactElement {
    return <div className="pf-v5-u-font-size-sm pf-v5-u-pt-sm">{children}</div>;
}

export default AccessControlDescription;
