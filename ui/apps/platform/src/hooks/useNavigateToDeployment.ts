import { useHistory } from 'react-router-dom';

type NavigateHook = (deploymentId: string, type: string) => void;
type NavigateToDeploymentById = () => void;

function useNavigateToDeployment(): NavigateHook {
    const history = useHistory();
    return function onNavigateToDeploymentById(
        deploymentId: string,
        type: string
    ): NavigateToDeploymentById {
        return function onNavigate(): void {
            if (type === 'external' || type === 'cidr') {
                history.push(`/main/network/${deploymentId}/${type}`);
                return;
            }
            history.push(`/main/network/${deploymentId}`);
        };
    };
}

export default useNavigateToDeployment;
