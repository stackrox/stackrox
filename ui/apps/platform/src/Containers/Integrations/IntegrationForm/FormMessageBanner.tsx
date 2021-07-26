import React, { ReactElement } from 'react';
import { Banner } from '@patternfly/react-core';

export type FormResponseMessage = {
    message: string;
    isError: boolean;
};

export type FormMessageBannerProps = {
    message: FormResponseMessage;
};

function FormMessageBanner({ message }: FormMessageBannerProps): ReactElement {
    return (
        <Banner variant={message.isError ? 'danger' : 'success'} className="pf-u-color-100">
            {message.message}
        </Banner>
    );
}

export default FormMessageBanner;
