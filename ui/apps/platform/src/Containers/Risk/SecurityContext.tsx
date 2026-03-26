import type { Deployment } from 'types/deployment.proto';
import SecurityContextCard from 'Components/SecurityContextCard';

type SecurityContextProps = {
    deployment: Deployment;
};

function SecurityContext({ deployment }: SecurityContextProps) {
    return <SecurityContextCard containers={deployment.containers} />;
}

export default SecurityContext;
