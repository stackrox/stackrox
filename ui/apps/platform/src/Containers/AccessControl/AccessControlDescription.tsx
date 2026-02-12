import { Content } from '@patternfly/react-core';
import type { ReactElement, ReactNode } from 'react';

export type AccessControlDescriptionProps = {
    children: ReactNode;
};

/*
 * Render description following AccessControlNav and preceding Title h2 in list or h1 in form element.
 */
function AccessControlDescription({ children }: AccessControlDescriptionProps): ReactElement {
    return <Content component="p">{children}</Content>;
}

export default AccessControlDescription;
