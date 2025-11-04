import type { FunctionComponent, PropsWithChildren } from 'react';

import type { IntegrationSource } from 'types/integration';

export type IntegrationsTabProps = {
    sourcesEnabled: IntegrationSource[];
};

export type IntegrationsTabElement = FunctionComponent<PropsWithChildren<IntegrationsTabProps>>;
