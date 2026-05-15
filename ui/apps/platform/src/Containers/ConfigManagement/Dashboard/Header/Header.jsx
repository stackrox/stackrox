import usePermissions from 'hooks/usePermissions';
import PoliciesTile from './PoliciesTile';
import CISControlsTile from './CISControlsTile';
import AppMenu from './AppMenu';
import RBACMenu from './RBACMenu';

const Header = () => {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForPoliciesTile = hasReadAccess('WorkflowAdministration');
    const hasReadAccessForCISControlsTile = hasReadAccess('Compliance');
    return (
        <div className="flex flex-1 justify-end">
            <div className="flex">
                {hasReadAccessForPoliciesTile && <PoliciesTile />}
                {hasReadAccessForCISControlsTile && <CISControlsTile />}
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
