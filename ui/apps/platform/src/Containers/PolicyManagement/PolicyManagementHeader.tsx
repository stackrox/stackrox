import TabNavHeader from 'Components/TabNav/TabNavHeader';
import { policiesBasePath, policyCategoriesPath } from 'routePaths';

type PolicyManagementHeaderProps = {
    currentTabTitle: string;
};

function PolicyManagementHeader({ currentTabTitle }: PolicyManagementHeaderProps) {
    const tabLinks = [
        { title: 'Policies', href: policiesBasePath },
        { title: 'Policy categories', href: policyCategoriesPath },
    ];

    return (
        <>
            <TabNavHeader
                currentTabTitle={currentTabTitle}
                tabLinks={tabLinks}
                pageTitle="Policy management - Policy categories"
                mainTitle="Policy management"
            />
        </>
    );
}

export default PolicyManagementHeader;
