import React, { useEffect, useState } from 'react';

import { integrationsPath } from 'routePaths';
import { fetchSignatureIntegrations } from 'services/SignatureIntegrationsService';
import { SignatureIntegration } from 'types/signatureIntegration.proto';
import tableColumnDescriptor from 'Containers/Integrations/utils/tableColumnDescriptor';
import TableModal from './TableModal';

function ImageSigningTableModal({ setValue, value, readOnly }) {
    const [integrations, setIntegrations] = useState<SignatureIntegration[]>([]);
    const rows = integrations.map((integration) => {
        return {
            ...integration,
            link: `${integrationsPath}/signatureIntegrations/signature/view/${integration.id}`,
        };
    });
    const columns = [...tableColumnDescriptor.signatureIntegrations.signature];

    useEffect(() => {
        fetchSignatureIntegrations()
            .then((data) => {
                setIntegrations(data);
            })
            .catch(() => {
                setIntegrations([]);
            });
    }, []);

    return (
        <TableModal
            typeText="trusted image signer"
            setValue={setValue}
            value={value}
            readOnly={readOnly}
            rows={rows}
            columns={columns}
        />
    );
}

export default ImageSigningTableModal;
