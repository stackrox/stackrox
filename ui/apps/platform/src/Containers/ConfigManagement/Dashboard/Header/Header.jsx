import usePermissions from 'hooks/usePermissions';
import PoliciesTile from './PoliciesTile';
import AppMenu from './AppMenu';
import RBACMenu from './RBACMenu';

const Header = () => {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForPoliciesTile = hasReadAccess('WorkflowAdministration');
    return (
        <div className="flex flex-1 justify-end">
            <div className="flex">
                {hasReadAccessForPoliciesTile && <PoliciesTile />}
                <div className="flex w-32 mr-2">
                    <AppMenu />
                </div>
                <div className="flex w-32 mr-3">
                    <RBACMenu />
                </div>
            </div>
        </div>
    );
};

export default Header;
