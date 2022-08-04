import { ReactElement } from 'react';

type TabProps = {
    title?: string;
    children: ReactElement;
};

// The "title" prop is necessary when used with the "useTabs" hook
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function Tab({ title, children }: TabProps): ReactElement {
    return children;
}

export default Tab;
