import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Button, Flex, FlexItem, ModalBoxBody, ModalBoxFooter } from '@patternfly/react-core';

import { ListPolicy } from 'types/policy.proto';
import { policiesBasePath } from 'routePaths';

type ImportPolicyJSONSuccessProps = {
    policies: ListPolicy[];
    handleCloseModal: () => void;
};

function ImportPolicyJSONSuccess({
    policies,
    handleCloseModal,
}: ImportPolicyJSONSuccessProps): ReactElement {
    return (
        <>
            <ModalBoxBody>
                The following
                {` ${policies.length === 1 ? 'policy has' : 'policies have'} `}
                been imported:
                <Flex
                    direction={{ default: 'column' }}
                    className="pf-u-mt-md"
                    data-testid="policies-imported"
                >
                    {policies.map(({ id, name }, idx) => (
                        <FlexItem key={id} className={idx === 0 ? '' : 'pf-u-pt-sm'}>
                            <Link to={`${policiesBasePath}/${id}`}>{name}</Link>
                        </FlexItem>
                    ))}
                </Flex>
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button key="close" variant="primary" onClick={handleCloseModal}>
                    Close
                </Button>
            </ModalBoxFooter>
        </>
    );
}

export default ImportPolicyJSONSuccess;
